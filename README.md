# Posty - Project 1

## Project
Group: Group01
Projectname: Posty
Groupmembers: Benedikt Lang

## Software, Tools, Libs

### Golang
The original suggestion was to use `java` but Prof. Reiser agreed on an approach in `Go`.
Golang was developed by google engineers to build large long-running systems and is especially suited for webservices. 

The benefits:

* Most code is more concise than in `java` 
* Lowlevel oauth2/oidc implementation possible (see later)
* Single binary deployment

### WGO (Go Workspaces)
[wgo](https://github.com/skelterjohn/wgo) is a small wrapper around the `go` toolchain to enable a standalone workspace with fixed dependencies. Therefore `posty` is build using `wgo build posty` instead of `go build posty`.

### Docker
The `EBS` Deployment uses docker, because it fits the picture and give greater flexibility. The project is build to one static binary (posty backend) and the frontend directory (html, javascript,..).

Since `EBS` supports native golang deployments, this would be possible too but it's more complicated to build the binary using the fixed dependencies (see later).

For convenience everthing can be build using docker, no need to install a `node.js` or `golang` environment (see later).

### AngularJS, Bootstrap (Frontend)
A proper REST API needs a suitable consumer. AngularJS is a good choice to build an interactive frontend consuming a REST API. The whole application logic is in `frontend/app/static/scripts/controllers/main.js`. The view in `views/`.

Bootstrap is used to please the eye at least a little bit.

#### Important libraries (Backend)

- [xhandler](https://github.com/rs/xhandler): HTTP Handler wrapper using [net/context](https://godoc.org/golang.org/x/net/context). Better flexibility between different routers, uses `context` to transport `sessions` etc.
- [goji](https://github.com/zenazn/goji): Minimalistic Webframework. Gojis http router was used for speed and support for url patterns like `/api/post/:id`.
- [gorilla/sessions](https://github.com/gorilla/sessions)/[gorilla/securecookie](https://github.com/gorilla/securecookie): Secure Cookies (transporting the `user_id`) use HMAC and Encryption to be tamper-proof. It enables to scale the application horizontally since no session store is needed.

## Design (Backend)

### Model
The model encapsulates the data store logic of the application. It's devided in `user` and `post` since those are the stored entities.
While the package `model`...


## Build, Test and Run

### Build using Docker

Requirements:
- Docker 1.7+

#### Build backend
```
# Build container for builds
./build/backend/buildcontainer.sh
# Build ./posty using container
./build/backend/build.sh 
```
Output: `./posty` binary

#### Build frontend
```
# Build container for builds
./build/backend/buildcontainer.sh
# Build ./posty using container
./build/backend/build.sh 
```
Output: `./frontend/dist`

#### Build staging

After you build the backend and the frontend, the following structure should be present:

- ./posty (go binary)
- ./frontend/dist (frontend files)

Now you can build a production ready container:

```
./build/staging/buildcontainer.sh
```
This will create the container: `blang/posty-staging`

### Build manually

#### Build backend

Requirements:
- Go 1.4+
Build using `wgo` (for vendoring):
```
## Get wgo
go get github.com/skelterjohn/wgo

## Restore dependencies
wgo restore 

## Build
wgo build posty
```

#### Build frontend
Requirements:
- Nodejs

```
cd frontend
npm install -g grunt-cli bower

## Install local dependencies
npm install

## Build
grunt build
```
Output: `./dist`

### Build Elastic Beanstalk zip

You can create a valid zip file for Elastic bean stalk:

```
./build/staging/create_ebs_zip.sh
```
This will create `posty-staging.zip`.

### Run locally

Requirements:
- dynamodb local running on http://127.0.0.1:8000/
- local golang dev environment
- built `./posty`

Setup a development environment for the 12-factor app:
```
export POSTY_OIDC_GOOGLE_CLIENT_ID=[YOUR DATA HERE]
export POSTY_OIDC_GOOGLE_CLIENT_SECRET=[YOUR DATA HERE]
export POSTY_OIDC_PAYPAL_CLIENT_ID=[YOUR DATA HERE]
export POSTY_OIDC_PAYPAL_CLIENT_SECRET=[YOUR DATA HERE]
export POSTY_DYNAMODB_ENDPOINT=http://127.0.0.1:8000/
## Setup aws credentials or use ~/.aws credentials
export AWS_REGION=us-west-2
export AWS_ENDPOINT=http://127.0.0.1:8000/
export AWS_PROFILE=dev
export AWS_ACCESS_KEY_ID=dev
export AWS_SECRET_ACCESS_KEY=dev
```

Run the integration tests. This will create the dynamodb tables `user` and `post` needed.
```
wgo test posty/model/awsdynamo/integrationtest -test.v -integration
```

Run Posty:
```
./posty -frontend-path "./frontend/dist"
```


