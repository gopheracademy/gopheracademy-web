+++
title = "Patchwork Toolkit - Lightweight Platform for the Network of Things"
date = 2014-12-02T18:00:00Z
author = ["Alexandr Krylovskiy", "Oleksandr Lobunets"]
+++

[Patchwork](http://patchwork-toolkit.github.io/) is a toolkit for connecting various devices into a network of things or, in a more broad case - Internet of Things (IoT). The main goal of creating this toolkit is to have a lightweight set of components that can help to quickly integrate different devices (e.g. Arduinos, RaspberryPI’s, Plugwise, etc) into a smart environment and expose specific devices’ capabilities as RESTful/SOAP/CoAP/MQTT/etc services and data streams.

# TL; DR;

Briefly, what that all above and especially further in this article means is shown on the image below.

![](images/pw-tldr.png)

What the Patchwork toolkit is all about can be expressed simple like this (considering you as a hacker/hobbyist): you take your favourite electronics (bunch of sensors, LED strip, robot-toys, etc), connect them to a pocket-size Linux box, install Patchwork, do some quick configuration and you get RESTful API, data streams using MQTT, directory of your services, discovery in the LAN using Bonjour and _a damn-sexy, open source real-time dashboard_ based on [Freeboard](https://github.com/Freeboard/freeboard). All you need is your creative and focus on implementation of your **idea, not infrastructure!**

# Why
<!--
*  Device integration: access to hardware resources via over the network (e.g., smart home/office scenarios)
*  Applications: discovery and communication with devices/resources
*  Patchwork toolkit: integration through configuration
*  Basic principles: KISS, DRY
-->
The Internet of Things (IoT) is causing a hype all over the Internet, further boosted by the maker movement and the [hardware renaissance](http://www.pubnub.com/blog/internet-of-things-hardware-renaissance/). However, implementing even basic IoT scenarios like smart home/office today is still challenging. One of the main challenges is to connect IoT devices (sensors, actuators) to the network and provide access to them for applications using common APIs and protocols.

The IoT devices market is growing, and it is very simple to build your own "sensor platform" for under $100 using a Raspberry Pi and a handful of [sensors](http://www.adafruit.com/categories/35). You don't even need to know much about embedded hardware to get started: following the [guides](https://learn.adafruit.com/category/raspberry-pi) and projects done by other people you can build a blinking prototype in just a few hours. In fact, you don't even need to know much about programming for embedded devices either: the same [guides](https://learn.adafruit.com/category/raspberry-pi) will walk you step-by-step through the process and provide with simple python examples and [libraries](https://github.com/adafruit/Adafruit-Raspberry-Pi-Python-Code) that make such prototyping very easy.

Once you have things working locally though, you start running into troubles: how to expose these devices on the network? how to discover and access them to build web/mobile applications to monitor and actuate things? At this point, you basically have two options:

1. Write a simple web/ws server and/or setup an [MQTT](http://mqtt.org) broker and publish to it (and hardcode endpoints)
2. Find an existing IoT framework/toolkit and integrate your devices/applications with it

Without going into much details, we got tired of doing 1. over and over again, and couldn't find 2. that would satisfy our expectations in being **simple**, **lightweight**, **easy to deploy and work with**. With these goals in mind, we started creating [Patchwork](http://patchwork-toolkit.github.io/) - a lightweight toolkit for IoT development that offers integration of devices through configuration and provides basic services for zeroconf discovery of resources and services on the network.

# Architecture

## Overview
A bird's-eye-view of the Patchwork architecture is shown in the picture:

![overview](images/pw-overview.png)

Patchwork integrates devices, applications and services with the help of the following components:

* **Device Gateway (DGW)** integrating different IoT devices and exposing their resources on the network via common APIs (REST, MQTT)
* **Device Catalog (DC)** providing a registry of available IoT devices and their capabilities on the network
* **Service Catalog (SC)** providing a registry of available services (MQTT broker, Device Catalog, DB, ...) on the network
 
## Device Gateway

<!--* Process manager: management of agents and stdin/stdout redirection
* Comm services: routing/proxying of requests and data streams
  * Extensible protocols, currently implemented: REST, MQTT
-->
A high-level architecture of the DGW capturing its main modules is shown in the picture:

![dgw](images/pw-dgw.png)

* **Devices** are IoT devices connected to the DGW host communicating their native protocols (Serial, ZigBee, etc) with Device Agents
* **Device Agents** are small programs running on the DGW and communicating through *stdin/stdout* with the Process Manager
* **Process Manager** manages the Device Agents (system processes) and forwards data between them and the communication Services
* **Services** expose the devices managed by Device Agents via common APIs (REST/MQTT) and forward requests/responses and data streams to the applications communicating with them

Device agents for Patchwork can be implemented in any programming language suitable for integration of a particular device and [example agents](https://github.com/patchwork-toolkit/agent-examples) are provided. Having a device agent, the integration of a new device reduces to describing its capabilities and parameters to the agent and communication protocols in a json configuration file. The device will be then registered in the Device Catalog and its resources exposed via configured APIs.

## Discovery of Devices and Services

In Patchwork, we distinguish between discovery of network services and IoT devices, which is implemented by the Device and Service catalogs correspondingly. The catalogs serve as registries for both Patchwork components and third-party applications and services and expose RESTful APIs.

Devices integrated with the DGW are automatically registered in its local Device Catalog, which can be used by applications to search for devices with specific capabilities/meta-information integrated with a specific DGW. In addition to that, a network-wide Device Catalog can be configured and populated with registrations of devices from all DGWs in the network. 

Similarly, Service Catalog provides a registry of services running on the network, and can be used by applications to search for services by their meta-information. For example, the network-wide Device Catalogs can be registered in one or many Service Catalogs to be discovered by applications.

To enable [zeroconf](http://en.wikipedia.org/wiki/Zero-configuration_networking) networking and enable discovery of services and IoT devices by applications without manually configuring the endpoints or IP addresses, we use [DNS-SD](http://dns-sd.org/) to advertise the Service Catalog endpoint on the network. Having discovered the Service Catalog, applications may proceed by querying it for available services and then discover devices by querying the Device Catalog.

# Implementation

## Technology evaluation

* Flaws in java, python solutions
* Why Go:
 * concurrency for process management
 * static linking (deployment on pocket-size computers)
 * cross-platform builds
 * performance
 * productivity
 * fun

## Highlights

* JSON configs
* Channels and diverse concurrency patterns
* Process management from forego
* Standard library
 * `net/http` implements most of the required functionality, only `gorilla/mux` router and `codegangsta/negroni` middleware for future extensions
 * `crypto/tls` surprisingly easy to use TLS (for MQTT)
 * Network stack for implementing (m)dns(-sd)
* `godep` for dependency management and vendoring
* Simple logging from multiple modules with `log`

# Usage example

## Dashboard out of the box

![Build-in Freeboard](images/pw-dashboard.png)

## Quick prototyping using IBM's NodeRed

![Data fusion using Device Gateway's API](images/pw-nodered-1.png)

![Audio and visual notifications](images/pw-nodered-2.png)

# Summay
Future work