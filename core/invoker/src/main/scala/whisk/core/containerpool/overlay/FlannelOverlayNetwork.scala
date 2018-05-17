package whisk.core.containerpool.overlay
import java.nio.file.{Files, Path, Paths}
import java.util.Properties

import akka.event.Logging.InfoLevel
import whisk.common.{Logging, LoggingMarkers, TransactionId}
import whisk.core.containerpool.docker.DockerApi

import scala.concurrent.{ExecutionContext, Future}

/**
  * Represents the subnet settings produced by Flannel
  * @param network the overall overlay network, in CIDR notation
  * @param subnet the subnet dedicated to this host
  * @param mtu the MTU to use with the network
  * @param ipmasq whether or not to use IP masquerading with this network
  */
case class FlannelSubnet(network: String, subnet: String, mtu: Int, ipmasq: Boolean)


class FlannelOverlayNetwork(override val name: String, config: FlannelSubnet, docker: DockerApi)(implicit ec: ExecutionContext) extends OverlayNetwork {
  override def destroy()(implicit transid: TransactionId): Future[Unit] = docker.removeNetwork(name)
}

object FlannelOverlayNetworkProvider extends OverlayNetworkProvider {
  private def loadSubnet(subnetFile: Path)(implicit transid: TransactionId, log: Logging): FlannelSubnet = {
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

  override def getOverlayNetwork(name: String, docker: DockerApi)(implicit ec: ExecutionContext, log: Logging): Future[OverlayNetwork] = {
    implicit val transid = TransactionId.invoker

    val config = loadSubnet(Paths.get("/run/flannel/subnet.env"))

    docker.removeNetwork(name).recover { case _ => () }.flatMap { _ =>
      docker.createNetwork(name, config.subnet, Map(
        "com.docker.network.bridge.enable_ip_masquerade" -> config.ipmasq.toString,
        "com.docker.network.driver.mtu" -> config.mtu.toString,
        "com.docker.network.bridge.name" -> "flannelbr0"
      )).map(_ => new FlannelOverlayNetwork(name, config, docker)) }
  }
}
