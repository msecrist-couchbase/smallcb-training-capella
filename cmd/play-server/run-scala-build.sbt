ThisBuild / scalaVersion := "2.12.12"
ThisBuild / organization := "com.couchbase"

lazy val example = (project in file("."))
  .settings(
    name := "Example",
    libraryDependencies += "com.couchbase.client" %% "scala-client" % "1.2.1",
  )