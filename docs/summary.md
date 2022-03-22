# go-vite summary

Table of Content:
* [Directories & packages](#directories)
  * [client](#client)
* [Appendix](#appendix)
  * [Setup goplantuml](#goplantuml)
  * [Other visualization tools](#visualization)

## Directories & packages <a name="directories"></a>

### client <a name="client"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/client.png" alt="client">
</p>

[[Full client diagram]](/docs/images/summary_diagrams/puml/client_full.png)

implements the go version client for remotely calling gvite nodes

## Appendix <a name="appendix"></a>

### Setup goplantuml <a name="goplantuml"></a>

### Other visualization tools <a name="visualization"></a>

#### GoCity

GoCity is an implementation of the Code City metaphor for visualizing source code. GoCity represents a Go program as a city, as follows:

* Folders are districts
* Files are buildings
* Structs are represented as buildings on the top of their files.

Structures Characteristics

* The Number of Lines of Source Code (LOC) represents the build color (high values makes the building dark)
* The Number of Variables (NOV) correlates to the building's base size.
* The Number of methods (NOM) correlates to the building height.

https://go-city.github.io/#/github.com/vitelabs/go-vite

<p align="center">
  <img src="/docs/images/summary_appendix/gocity.gif" alt="gocity">
</p>