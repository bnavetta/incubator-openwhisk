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

import java.io.FileNotFoundException
import java.nio.file.Files
import java.nio.file.Paths
import java.util.concurrent.Semaphore

import akka.actor.ActorSystem

import scala.collection.concurrent.TrieMap
import scala.concurrent.blocking
import scala.concurrent.ExecutionContext
import scala.concurrent.Future
import scala.util.Failure
import scala.util.Success
import scala.util.Try
import akka.event.Logging.{ErrorLevel, InfoLevel}
import pureconfig.loadConfigOrThrow
import whisk.common.Logging
import whisk.common.LoggingMarkers
import whisk.common.TransactionId
import whisk.core.ConfigKeys
import whisk.core.containerpool.ContainerId
import whisk.core.containerpool.ContainerAddress

import scala.concurrent.duration.Duration

object DockerContainerId {

  val containerIdRegex = """^([0-9a-f]{64})$""".r

  def parse(id: String): Try[ContainerId] = {
    id match {
      case containerIdRegex(_) => Success(ContainerId(id))
      case _                   => Failure(new IllegalArgumentException(s"Does not comply with Docker container ID format: ${id}"))
    }
  }
}

/**
 * Configuration for docker client command timeouts.
 */
case class DockerClientTimeoutConfig(run: Duration,
                                     rm: Duration,
                                     pull: Duration,
                                     ps: Duration,
                                     pause: Duration,
                                     unpause: Duration,
                                     inspect: Duration,
                                     network: Duration)

/**
 * Serves as interface to the docker CLI tool.
 *
 * Be cautious with the ExecutionContext passed to this, as the
 * calls to the CLI are blocking.
 *
 * You only need one instance (and you shouldn't get more).
 */
class DockerClient(dockerHost: Option[String] = None,
                   timeouts: DockerClientTimeoutConfig =
                     loadConfigOrThrow[DockerClientTimeoutConfig](ConfigKeys.dockerTimeouts))(
  executionContext: ExecutionContext)(implicit log: Logging, as: ActorSystem)
    extends DockerApi
    with ProcessRunner {
  implicit private val ec = executionContext

  // Determines how to run docker. Failure to find a Docker binary implies
  // a failure to initialize this instance of DockerClient.
  protected val dockerCmd: Seq[String] = {
    val alternatives = List("/usr/bin/docker", "/usr/local/bin/docker")

    val dockerBin = Try {
      alternatives.find(a => Files.isExecutable(Paths.get(a))).get
    } getOrElse {
      throw new FileNotFoundException(s"Couldn't locate docker binary (tried: ${alternatives.mkString(", ")}).")
    }

    val host = dockerHost.map(host => Seq("--host", s"tcp://$host")).getOrElse(Seq.empty[String])
    Seq(dockerBin) ++ host
  }

  protected val maxParallelRuns = 10
  protected val runSemaphore = new Semaphore( /* permits= */ maxParallelRuns, /* fair= */ true)

  // Docker < 1.13.1 has a known problem: if more than 10 containers are created (docker run)
  // concurrently, there is a good chance that some of them will fail.
  // See https://github.com/moby/moby/issues/29369
  // Use a semaphore to make sure that at most 10 `docker run` commands are active
  // the same time.
  def run(image: String, args: Seq[String] = Seq.empty[String])(
    implicit transid: TransactionId): Future[ContainerId] = {
    Future {
      blocking {
        // Acquires a permit from this semaphore, blocking until one is available, or the thread is interrupted.
        // Throws InterruptedException if the current thread is interrupted
        runSemaphore.acquire()
      }
    }.flatMap { _ =>
      // Iff the semaphore was acquired successfully
      runCmd(Seq("run", "-d") ++ args ++ Seq(image), timeouts.run)
        .andThen {
          // Release the semaphore as quick as possible regardless of the runCmd() result
          case _ => runSemaphore.release()
        }
        .map(ContainerId.apply)
        .recoverWith {
          // https://docs.docker.com/v1.12/engine/reference/run/#/exit-status
          // Exit code 125 means an error reported by the Docker daemon.
          // Examples:
          // - Unrecognized option specified
          // - Not enough disk space
          case pre: ProcessRunningException if pre.exitCode == 125 =>
            Future.failed(
              DockerContainerId
                .parse(pre.stdout)
                .map(BrokenDockerContainer(_, s"Broken container: ${pre.getMessage}"))
                .getOrElse(pre))
        }
    }
  }

  def inspectIPAddress(id: ContainerId, network: String)(implicit transid: TransactionId): Future[ContainerAddress] =
    runCmd(
      Seq("inspect", "--format", s"{{.NetworkSettings.Networks.${network}.IPAddress}}", id.asString),
      timeouts.inspect).flatMap {
      case "<no value>" => Future.failed(new NoSuchElementException)
      case stdout       => Future.successful(ContainerAddress(stdout))
    }

  def pause(id: ContainerId)(implicit transid: TransactionId): Future[Unit] =
    runCmd(Seq("pause", id.asString), timeouts.pause).map(_ => ())

  def unpause(id: ContainerId)(implicit transid: TransactionId): Future[Unit] =
    runCmd(Seq("unpause", id.asString), timeouts.unpause).map(_ => ())

  def rm(id: ContainerId)(implicit transid: TransactionId): Future[Unit] =
    runCmd(Seq("rm", "-f", id.asString), timeouts.rm).map(_ => ())

  def ps(filters: Seq[(String, String)] = Seq.empty, all: Boolean = false)(
    implicit transid: TransactionId): Future[Seq[ContainerId]] = {
    val filterArgs = filters.flatMap { case (attr, value) => Seq("--filter", s"$attr=$value") }
    val allArg = if (all) Seq("--all") else Seq.empty[String]
    val cmd = Seq("ps", "--quiet", "--no-trunc") ++ allArg ++ filterArgs
    runCmd(cmd, timeouts.ps).map(_.lines.toSeq.map(ContainerId.apply))
  }

  override def createNetwork(name: String, subnet: String, options: Map[String, String])(implicit transid: TransactionId): Future[Unit] = {
    val optArgs = if (options.isEmpty) Seq.empty else Seq("-o", options.map{ case (k, v) => s"$k=$v" }.mkString(","))
    runCmd(Seq("network", "create", name, "--subnet", subnet) ++ optArgs, timeouts.network).map(_ => ())
  }

  override def removeNetwork(name: String)(implicit transid: TransactionId): Future[Unit] = runCmd(Seq("network", "rm", name), timeouts.network).map(_ => ())

  override def connect(id: ContainerId, network: String)(implicit transid: TransactionId): Future[Unit] = runCmd(Seq("network", "connect", network, id.asString), timeouts.network).map(_ => ())

  override def disconnect(id: ContainerId, network: String)(implicit transid: TransactionId): Future[Unit] = runCmd(Seq("network", "disconnect", network, id.asString), timeouts.network).map(_ => ())

  /**
   * Stores pulls that are currently being executed and collapses multiple
   * pulls into just one. After a pull is finished, the cached future is removed
   * to enable constant updates of an image without changing its tag.
   */
  private val pullsInFlight = TrieMap[String, Future[Unit]]()
  def pull(image: String)(implicit transid: TransactionId): Future[Unit] =
    pullsInFlight.getOrElseUpdate(image, {
      runCmd(Seq("pull", image), timeouts.pull).map(_ => ()).andThen { case _ => pullsInFlight.remove(image) }
    })

  def isOomKilled(id: ContainerId)(implicit transid: TransactionId): Future[Boolean] =
    runCmd(Seq("inspect", id.asString, "--format", "{{.State.OOMKilled}}"), timeouts.inspect).map(_.toBoolean)

  private def runCmd(args: Seq[String], timeout: Duration)(implicit transid: TransactionId): Future[String] = {
    val cmd = dockerCmd ++ args
    val start = transid.started(
      this,
      LoggingMarkers.INVOKER_DOCKER_CMD(args.head),
      s"running ${cmd.mkString(" ")} (timeout: $timeout)",
      logLevel = InfoLevel)
    executeProcess(cmd, timeout).andThen {
      case Success(_) => transid.finished(this, start)
      case Failure(t) => transid.failed(this, start, t.getMessage, ErrorLevel)
    }
  }
}

trait DockerApi {

  /**
   * Spawns a container in detached mode.
   *
   * @param image the image to start the container with
   * @param args arguments for the docker run command
   * @return id of the started container
   */
  def run(image: String, args: Seq[String] = Seq.empty[String])(implicit transid: TransactionId): Future[ContainerId]

  /**
   * Gets the IP address of a given container.
   *
   * A container may have more than one network. The container has an
   * IP address in each of these networks such that the network name
   * is needed.
   *
   * @param id the id of the container to get the IP address from
   * @param network name of the network to get the IP address from
   * @return ip of the container
   */
  def inspectIPAddress(id: ContainerId, network: String)(implicit transid: TransactionId): Future[ContainerAddress]

  /**
   * Pauses the container with the given id.
   *
   * @param id the id of the container to pause
   * @return a Future completing according to the command's exit-code
   */
  def pause(id: ContainerId)(implicit transid: TransactionId): Future[Unit]

  /**
   * Unpauses the container with the given id.
   *
   * @param id the id of the container to unpause
   * @return a Future completing according to the command's exit-code
   */
  def unpause(id: ContainerId)(implicit transid: TransactionId): Future[Unit]

  /**
   * Removes the container with the given id.
   *
   * @param id the id of the container to remove
   * @return a Future completing according to the command's exit-code
   */
  def rm(id: ContainerId)(implicit transid: TransactionId): Future[Unit]

  /**
   * Returns a list of ContainerIds in the system.
   *
   * @param filters Filters to apply to the 'ps' command
   * @param all Whether or not to return stopped containers as well
   * @return A list of ContainerIds
   */
  def ps(filters: Seq[(String, String)] = Seq.empty, all: Boolean = false)(
    implicit transid: TransactionId): Future[Seq[ContainerId]]

  /**
   * Pulls the given image.
   *
   * @param image the image to pull
   * @return a Future completing once the pull is complete
   */
  def pull(image: String)(implicit transid: TransactionId): Future[Unit]

  /**
   * Determines whether the given container was killed due to
   * memory constraints.
   *
   * @param id the id of the container to check
   * @return a Future containing whether the container was killed or not
   */
  def isOomKilled(id: ContainerId)(implicit transid: TransactionId): Future[Boolean]

  /**
    * Creates a bridge network
    *
    * @param name the name of the network (within Docker, not necessarily the bridge interface name)
    * @param subnet the subnet, in CIDR format, that this network covers
    * @param options additional options to the bridge driver
    * @return a Future completing once the network is created
    */
  def createNetwork(name: String, subnet: String, options: Map[String, String] = Map.empty)(implicit transid: TransactionId): Future[Unit]

  /**
    * Removes the Docker network with the given name
    *
    * @param name the network name
    * @return a Future completing once the network has been destroyed
    */
  def removeNetwork(name: String)(implicit transid: TransactionId): Future[Unit]

  /**
    * Connect a container to a network
    *
    * @param id the id of the container to connect
    * @param network the network to attach the container to
    * @return a Future completing once the container is connected
    */
  def connect(id: ContainerId, network: String)(implicit transid: TransactionId): Future[Unit]

  /**
    * Disconnect a container from a network
    *
    * @param id the id of the container to disconnect
    * @param network the network to remove the container from
    * @return a Future completing once the container is disconnected
    */
  def disconnect(id: ContainerId, network: String)(implicit transid: TransactionId): Future[Unit]
}

/** Indicates any error while starting a container that leaves a broken container behind that needs to be removed */
case class BrokenDockerContainer(id: ContainerId, msg: String) extends Exception(msg)
