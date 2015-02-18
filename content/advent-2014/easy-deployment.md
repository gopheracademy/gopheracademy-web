+++
author = ["Brian Ketelsen"]
date = "2014-12-10T08:00:00+00:00"
title = "Easy Docker Deployment with Hooks and Captain Hook"
series = ["Advent 2014"]
+++

## Deploying with "git push" the Docker Way

Many people have asked me how we set up the GopherAcademy blog to automatically deploy when we push a commit.  In this Go Advent 2014 article I'm going to walk through the process so you can see what is involved and decide if it's right for your setup.

### Why

Deployment can be the hardest part of any project.  Docker certainly makes that step easier but the ecosystem is still young, and if you want a smooth workflow you've got to patch a few things together yourself.  There are projects like Dokku at the lower end and Kubernetes at the higher end that do much of this work for you, but for the GopherAcademy setup we need more than Dokku does and much less than Kubernetes can do.

### Docker-ize your Application

The first step of this process is to create a Docker container that can run your application.  Since the GopherAcademy blog runs on [hugo](https://github.com/spf13/hugo) we need to take that into account.  I started with a base Docker container I borrowed from [tutum](https://github.com/tutumcloud) and then [heavily modified it](https://github.com/bketelsen/hugo-nginx-base) for our needs.  All containers that we deploy with Hugo websites use this container as the base.

The base Dockerfile looks like this:

```
FROM ubuntu:trusty
MAINTAINER Feng Honglin <hfeng@tutum.co>

RUN apt-get update && \
    apt-get install -y nginx && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN apt-get update -y && apt-get install --no-install-recommends -y -q curl build-essential ca-certificates git mercurial bzr
RUN mkdir /goroot && curl https://storage.googleapis.com/golang/go1.3.1.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1
RUN mkdir /gopath

ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

RUN go get -v github.com/spf13/hugo
RUN go install github.com/spf13/hugo

ONBUILD ADD . /site-source
ONBUILD RUN cd /site-source && \
	hugo

ONBUILD RUN cp -R /site-source/public /app/


RUN echo "daemon off;" >> /etc/nginx/nginx.conf
ADD sites-enabled/ /etc/nginx/sites-enabled/
#ADD app/ /app/

EXPOSE 80

CMD ["/usr/sbin/nginx"]
```

The important thing to note here is the use of the `ONBUILD` directives, which defer build actions to run later, when a container built with this container as a base is built.  The `ONBUILD` actions here will add the current directory as `site-source` to the container, then run `hugo` to generate the website from the `site-source` folder.  The rest of the dockerfile is just plumbing to get nginx working.

In the Docker file for this blog there's very little to show:

```
FROM bketelsen/hugo-nginx-docker
ADD sites-enabled/ /etc/nginx/sites-enabled/
```

That's because all the hard work was done in the base container.  So any new website we do using Hugo will follow this same pattern.  The nginx configuration will be stored in a folder called `sites-enabled` which will be added to the Docker container and overwrite the default nginx settings of the base container.  The base container will generate the HTML when the Docker image is built, putting it exactly where nginx expects it to be.

### Automated Builds

The next step of this flow is to create an automated build in the Docker Hub.  After logging in to [the docker hub](https://hub.docker.com), click on the button that says "Add Repository", and choose "Trusted Build".  You'll need to link your docker hub account to your github account.  Then choose the repository that contains the dockerfile that you want to build.  Because it's a `Trusted Build` it will only build the container when you commit changes to Github.  Now you've got a workflow that generates docker builds every time you make a change to your repository.

To hook this up to deployment, I created a quick little webhook listener called [captainhook](https://github.com/bketelsen/captainhook).  It does only one useful thing: it runs a script when it receives a post from a webhook.  To install it `go get github.com/bketelsen/captainhook`.  Then create a config directory.  Mine is called, appropriately, `captainhook`.  I put it in my user's home directory.

Now you'll need a configuration file in that directory.  You'll want one for each action you want to trigger remotely.  Since we're going to configure Docker Hub to call us when the automated build is done, I created a config file called `gablog.json` in that configuration directory.  

Here's the configuration file:

```
{
    "scripts": [
        {
            "command": "/root/gablog.sh",
            "args": [
                "3"
            ]
        }
    ]
}

```
All we've done is tell `captainhook` to run a script called `gablog.sh` in the /root directory with `3` as an argument.

Here's that script:

```
# /root/gablog.sh

if [ -z "$1" ]
  then
    echo "usage : gablog.sh 3 -- start three new instances"
	exit -1
fi


echo "Getting currently running gablog containers"
OLDPORTS=( `docker ps | grep gopheracademy-web | awk '{print $1}'` )
echo "pulling new version"

docker pull bketelsen/gopheracademy-web
echo "starting new containers"
for i in `seq 1 $1` ; do
	docker run -d -e VIRTUAL_HOST=blog.gopheracademy.com -p 80 bketelsen/gopheracademy-web 
done

echo "removing old containers"
for i in ${OLDPORTS[@]} 
do
	echo "removing old container $i"
	docker kill $i 
done

```
This script isn't too complicated.  It looks for the container-id's of the running containers, storing them for later.  Then it starts up new containers, and kills the old ones.  The interesting part is the environment variable `VIRTUAL_HOST` which we'll see a bit later in this process.

Now start captainhook with the configuration directory specified on the command line:

`$> captainhook -listen-addr=0.0.0.0:8080 -echo -configdir /root/captainhook &`

You can test it with curl:

`$> curl http://127.0.0.1:8080/gablog.json`

You should see a simple response from the server, and if you run `docker ps` you should see three containers running.  If you don't, make sure you made the `gablog.sh` script executable with `chmod +x gablog.sh`.  

So now we have something ready for a webhook on the deployment server, go back to Docker hub and configure a webhook that corresponds to where `captainhook` is hosted.  In my case, it's running on the blog.gopheracademy.com server, so my webhook URL is `http://blog.gopheracademy.com/gablog`.  Captainhook will look for the config file called `gablog.json`, and execute whatever shell script is listed there.  Read the documentation for `captainhook` on Github to understand why it's so limited.  It all boils down to security.

Our deployment process looks like this:

```
git commit -m"new article"
  ---->
hub.docker.com builds the docker container and calls the configured webhook
  ---->
captainhook runs the bash script associated with the webhook
  ---->
bash script starts new containers, stops old one.  Site is Deployed!
```

The only missing piece here is a web proxy to wrap those 3 instances of the GopherAcademy blog into a single listener.  I found a docker container called `jwilder/nginx-proxy` that does just this.  As long as you start your docker containers with that `VIRTUAL_HOST` environment variable we showed earlier, the `nginx-proxy` container will proxy all VHOST requests for that host to your containers in round-robin format.  That means you can have one `nginx-proxy` container running and any number of other websites all running in docker proxied by that container.  We host the GopherAcademy main site, the GopherCon site, the GopherAcademy Blog and at least 4 other sites all behind that single nginx proxy. 

###  Wrapup
While it's not *directly* related to Go, it's still a fun little project that will let you automatically deploy your websites from a git push.  Captainhook is written in Go, and of course, so is Docker.  So there is some relevance to the Go Advent Calendar here somewhere, I promise.  This setup has been running all of our websites on a single [Digital Ocean(with our referral code)](https://www.digitalocean.com/?refcode=9dd266a276e6) droplet for months without a single problem (not counting the one time I screwed everything up manually).  The entire process takes very little time, and depends on how busy the Docker Hub build servers are.  Typically it's all done in less than 5 minutes, and it requires no manual intervention at all.  That's my kind of devops.
