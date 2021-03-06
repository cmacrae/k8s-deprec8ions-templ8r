<p align="center">
  <b>k8s-deprec8ions-templ8r</b>
</p>
<p align="center">
  <a href="https://github.com/cmacrae/k8s-deprec8ions-templ8r/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/cmacrae/k8s-deprec8ions-templ8r.svg?color=a6dcef" alt="License Badge">
  </a>
  <a href="https://github.com/cmacrae/k8s-deprec8ions-templ8r/compare/v0.1.0...HEAD">
    <img src="https://img.shields.io/github/commits-since/cmacrae/k8s-deprec8ions-templ8r/latest.svg?color=ea907a" alt="Version Badge">
  </a>
  <a href="https://goreportcard.com/report/github.com/cmacrae/k8s-deprec8ions-templ8r">
    <img src="https://goreportcard.com/badge/github.com/cmacrae/k8s-deprec8ions-templ8r" alt="Go Report Card">
  </a>
</p>
<p align="center">
  <a href="https://github.com/users/cmacrae/packages/container/package/k8s-deprec8ions-templ8r">
    <img src="https://img.shields.io/badge/GHCR-image-87DCC0.svg?logo=GitHub" alt="GHCR Badge">
  </a>
  <a href="https://hub.docker.com/r/cmacrae/k8s-deprec8ions-templ8r">
    <img src="https://img.shields.io/badge/DockerHub-image-2496ED.svg?logo=Docker" alt="DockerHub Badge">
  </a>
  <a href="https://opencontainers.org/">
    <img src="https://img.shields.io/badge/OCI-compliant-262261.svg?logo=open-containers-initiative" alt="OCI Badge">
  </a>
  <a href="https://snyk.io">
    <img src="https://img.shields.io/badge/Snyk-protected-4C4A73.svg?logo=snyk" alt="Snyk Badge">
  </a>
</p>

#

> Well that's a stupid name...  

Yes, yes it is!


## About
A simple tool to render [Go templates](https://golang.org/pkg/text/template/) from deprecations in the Kubernetes API schema.  
Useful for crafting things like [Rego policies](https://www.openpolicyagent.org) to flag deprecations.

It works by downloading the [Swagger](https://swagger.io/) file from the Kubernetes repo (version of your choice)
and inspecting each API/API property's `description` field for occurrences of "deprecated". It then provides
some abstract objects you can build Go templates around to render this information as you please.

## Usage
```
k8s-deprec8ions-templ8r -version v1.20.2 -template templates/kove.rego.gotmpl
```

or with the [Docker image](https://github.com/users/cmacrae/packages/container/package/k8s-deprec8ions-templ8r)
```
docker run --rm -it \
    -v $(pwd)/templates:/kdt/templates \
    ghcr.io/cmacrae/k8s-deprec8ions-templ8r:v0.1.0 \
    -path swagger -version v1.20.2 -template templates/kove.rego.gotmpl
```
*Note: You must use `-path swagger` with this image*

### Options
```
  -force
    	Whether to force download the Kubernetes API Swagger file
  -log-level string
    	Log level. Should be: debug, info, warn, error (default "info")
  -path string
    	Path to read/download the Kubernetes API Swagger file (default same as 'version')
  -template string
    	Path to the template to render
  -version string
    	Kubernetes version to check for deprecations (default "master")
```

## Implementations
Some example templates can be found in the [`templates/`](templates) directory.  
The [kove template](tempaltes/k8s-deprec8ions-templ8r.rego) is used to provide policies for the [kove](https://github.com/cmacrae/kove) project: [kove-deprecations](https://artifacthub.io/packages/helm/cmacrae/kove-deprecations)

## Acknowledgements/Attributions
The meat of this code is based off [kubepug](https://github.com/rikatz/kubepug), so a big thank you to [@rikatz](https://github.com/rikatz)
for his work on a great tool!
