/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk.core.containerpool.docker

import akka.actor.ActorSystem

import scala.concurrent.Await
import scala.concurrent.ExecutionContext
import scala.concurrent.Future
import whisk.common.Logging
import whisk.common.TransactionId
import whisk.core.WhiskConfig
import whisk.core.containerpool.Container
import whisk.core.containerpool.ContainerFactory
import whisk.core.containerpool.ContainerFactoryProvider
import whisk.core.entity.ByteSize
import whisk.core.entity.ExecManifest
import whisk.core.entity.InstanceId

import scala.concurrent.duration._
import java.util.concurrent.TimeoutException

import pureconfig._
import whisk.core.ConfigKeys
import whisk.core.containerpool.ContainerArgsConfig
import whisk.core.containerpool.overlay.OverlayNetwork
import whisk.core.containerpool.overlay.FlannelOverlayNetworkProvider

case class DockerContainerFactoryConfig(useRunc: Boolean)

class DockerContainerFactory(instance: InstanceId,
                             parameters: Map[String, Set[String]],
                             containerArgsConfig: ContainerArgsConfig =
                               loadConfigOrThrow[ContainerArgsConfig](ConfigKeys.containerArgs),
                             dockerContainerFactoryConfig: DockerContainerFactoryConfig =
                               loadConfigOrThrow[DockerContainerFactoryConfig](ConfigKeys.dockerContainerFactory))(
  implicit actorSystem: ActorSystem,
  ec: ExecutionContext,
  logging: Logging,
  docker: DockerApiWithFileAccess,
  runc: RuncApi)
    extends ContainerFactory {

  private val overlayNetwork: Future[OverlayNetwork] = FlannelOverlayNetworkProvider.getOverlayNetwork("whisknet", docker)

  /** Create a container using docker cli */
  override def createContainer(tid: TransactionId,
                               name: String,
                               actionImage: ExecManifest.ImageName,
                               userProvidedImage: Boolean,
                               memory: ByteSize,
                               cpuShares: Int)(implicit config: WhiskConfig, logging: Logging): Future[Container] = {
    val image = if (userProvidedImage) {
      actionImage.publicImageName
    } else {
      actionImage.localImageName(config.dockerRegistry, config.dockerImagePrefix, Some(config.dockerImageTag))
    }

    for {
      container <- DockerContainer.create(
        tid,
        image = image,
        userProvidedImage = userProvidedImage,
        memory = memory,
        cpuShares = cpuShares,
        environment = Map("__OW_API_HOST" -> config.wskApiHost),
        network = containerArgsConfig.network,
        dnsServers = containerArgsConfig.dnsServers,
        name = Some(name),
        useRunc = dockerContainerFactoryConfig.useRunc,
        parameters ++ containerArgsConfig.extraArgs.map { case (k, v) => ("--" + k, v) })
      overlay <- overlayNetwork
      _ <- container.connect(overlay.name)(tid)
    } yield container
  }

  /** Perform cleanup on init */
  override def init(): Unit = removeAllActionContainers()

  /** Perform cleanup on exit - to be registered as shutdown hook */
  override def cleanup(): Unit = {
    implicit val transid = TransactionId.invoker
    try {
      removeAllActionContainers()

      overlayNetwork.onSuccess { case net => net.destroy() }
    } catch {
      case e: Exception => logging.error(this, s"Failed to remove action containers: ${e.getMessage}")
    }
  }

  /**
   * Removes all wsk_ containers - regardless of their state
   *
   * If the system in general or Docker in particular has a very
   * high load, commands may take longer than the specified time
   * resulting in an exception.
   *
   * There is no checking whether container removal was successful
   * or not.
   *
   * @throws InterruptedException     if the current thread is interrupted while waiting
   * @throws TimeoutException         if after waiting for the specified time this `Awaitable` is still not ready
   */
  @throws(classOf[TimeoutException])
  @throws(classOf[InterruptedException])
  private def removeAllActionContainers(): Unit = {
    implicit val transid = TransactionId.invoker
    val cleaning =
      docker.ps(filters = Seq("name" -> s"${ContainerFactory.containerNamePrefix(instance)}_"), all = true).flatMap {
        containers =>
          logging.info(this, s"removing ${containers.size} action containers.")
          val removals = containers.map { id =>
            (if (dockerContainerFactoryConfig.useRunc) {
               runc.resume(id)
             } else {
               docker.unpause(id)
             })
              .recoverWith {
                // Ignore resume failures and try to remove anyway
                case _ => Future.successful(())
              }
              .flatMap { _ =>
                docker.rm(id)
              }
          }
          Future.sequence(removals)
      }
    Await.ready(cleaning, 30.seconds)
  }
}

object DockerContainerFactoryProvider extends ContainerFactoryProvider {
  override def getContainerFactory(actorSystem: ActorSystem,
                                   logging: Logging,
                                   config: WhiskConfig,
                                   instanceId: InstanceId,
                                   parameters: Map[String, Set[String]]): ContainerFactory = {

    new DockerContainerFactory(instanceId, parameters)(
      actorSystem,
      actorSystem.dispatcher,
      logging,
      new DockerClientWithFileAccess()(actorSystem.dispatcher)(logging, actorSystem),
      new RuncClient()(actorSystem.dispatcher)(logging, actorSystem))
  }

}
