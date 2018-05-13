package whisk.core.containerpool.overlay

import whisk.common.{Logging, TransactionId}
import whisk.core.containerpool.docker.DockerApi

import scala.concurrent.{ExecutionContext, Future}

/**
  * Represents an overlay network that Docker worker containers can attach to.
  */
trait OverlayNetwork {
  /**
    * The name of the network, as used by Docker.
    */
  val name: String

  /**
    * Removes any resources, including the Docker network, associated with this overlay.
    * @return a Future which completes once the network has been shut down
    */
  def destroy()(implicit transid: TransactionId): Future[Unit]
}

trait OverlayNetworkProvider {
  def getOverlayNetwork(name: String, docker: DockerApi)(implicit ec: ExecutionContext, log: Logging): Future[OverlayNetwork]
}