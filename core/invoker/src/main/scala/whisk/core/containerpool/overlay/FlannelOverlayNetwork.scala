package whisk.core.containerpool.overlay
import whisk.common.{Logging, TransactionId}
import whisk.core.containerpool.docker.DockerApi

import scala.concurrent.{ExecutionContext, Future}

class FlannelOverlayNetwork(override val name: String, instance: FlannelInstance, docker: DockerApi)(implicit ec: ExecutionContext) extends OverlayNetwork {
  override def destroy()(implicit transid: TransactionId): Future[Unit] = Future.sequence(Seq(Future(instance.shutdown()), docker.removeNetwork(name))).map(_ => ())
}

object FlannelOverlayNetworkProvider extends OverlayNetworkProvider {
  override def getOverlayNetwork(name: String, docker: DockerApi)(implicit ec: ExecutionContext, log: Logging): Future[OverlayNetwork] = for {
    instance <- FlannelInstance.create(s"/run/flannel/$name", s"/openwhisk/flannel/$name")(ec, log, TransactionId.invoker)
    config = instance.config
    _ <- docker.createNetwork(name, config.subnet, Map(
      "com.docker.network.bridge.enable_ip_masquerade" -> config.ipmasq.toString,
      "com.docker.network.driver.mtu" -> config.mtu.toString
    ))(TransactionId.invoker) // TODO: is this the right TransactionId?
  } yield new FlannelOverlayNetwork(name, instance, docker)
}
