ThisBuild / scalaVersion := "2.12.12"
ThisBuild / organization := "com.couchbase"

lazy val example = (project in file("."))
  .settings(
    name := "Example",
    libraryDependencies += "com.couchbase.client" %% "scala-client" % "1.1.2",
    libraryDependencies += "org.slf4j" % "slf4j-log4j12" % "1.7.30",
  )