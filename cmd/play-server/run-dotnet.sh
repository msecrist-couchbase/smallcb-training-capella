#!/bin/bash

cd $(dirname ${1})

# csproj file, among other things, determines which packages to pull from nuget
echo '<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>netcoreapp3.1</TargetFramework></PropertyGroup><ItemGroup><PackageReference Include="CouchbaseNetClient" Version="3.2.2" /></ItemGroup></Project>' > project.csproj

# this file extension shouldn't be 'dotnet' in the first place
# Simple hack to workaround for JIRA:NCBC-2945
sed 's/localhost/127.0.0.1/g' code.dotnet > Program.cs
#cp code.dotnet Program.cs

# run/build
dotnet run
