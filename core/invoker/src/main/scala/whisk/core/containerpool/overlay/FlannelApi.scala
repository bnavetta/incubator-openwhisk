package whisk.core.containerpool.overlay

import java.nio.file.{Files, Path, Paths}
import java.util.Properties

import akka.event.Logging.InfoLevel
import whisk.common.{Logging, LoggingMarkers, TransactionId}

import scala.concurrent.{ExecutionContext, Future, blocking}
import scala.sys.process._

/**
  * Represents the subnet settings produced by Flannel
  * @param network the overall overlay network, in CIDR notation
  * @param subnet the subnet dedicated to this host
  * @param mtu the MTU to use with the network
  * @param ipmasq whether or not to use IP masquerading with this network
  */
case class FlannelSubnet(network: String, subnet: String, mtu: Int, ipmasq: Boolean)

class FlannelInstance(process: Process, subnetFile: Path)(implicit log: Logging, transid: TransactionId) {
  lazy val config: FlannelSubnet = loadSubnet()

  private def loadSubnet(): FlannelSubnet = {
    val start = transid.started(this, LoggingMarkers.INVOKER_FLANNEL, s"loading subnet configuration from $subnetFile", logLevel = InfoLevel)

    val props = new Properties()
    val is = Files.newInputStream(subnetFile)
    try {
      props.load(is)
    } finally {
      is.close()
    }

    val subnet = FlannelSubnet(
      network = props.getProperty("FLANNEL_NETWORK"),
      subnet = props.getProperty("FLANNEL_SUBNET"),
      mtu = props.getProperty("FLANNEL_MTU").toInt,
      ipmasq = props.getProperty("FLANNEL_IPMASQ").toBoolean
    )

    transid.finished(this, start, s"loaded subnet $subnet", logLevel = InfoLevel)

    subnet
  }

  def shutdown(): Unit = {
    val start = transid.started(this, LoggingMarkers.INVOKER_FLANNEL, "stopping flanneld", logLevel = InfoLevel)
    process.destroy()
    transid.finished(this, start, "stopped flanneld", logLevel = InfoLevel)
  }
}

object FlannelInstance {
  def create(subnetFile: String = "/run/flannel/openwhisk.env", etcdPrefix: String = "/openwhisk/flannel")(implicit ec: ExecutionContext, log: Logging, transid: TransactionId): Future[FlannelInstance] = {
    Future(blocking {
      val args = Seq("/usr/bin/flanneld", "-subnet-file", subnetFile, "-etcd-prefix", etcdPrefix)
      val start = transid.started(this, LoggingMarkers.INVOKER_FLANNEL, s"running ${args.mkString(" ")}", logLevel = InfoLevel)
      try {
        val process = args.run()
        transid.finished(this, start, "started flanneld", logLevel = InfoLevel)

        // TODO: poll for subnet file to be created

        return Future.successful(new FlannelInstance(process, Paths.get(subnetFile)))
      } catch {
        case e: Throwable =>
          transid.failed(this, start, s"failed to start flanneld: ${e.getMessage}", logLevel = InfoLevel)
          return Future.failed(e)
      }
    })
  }
}