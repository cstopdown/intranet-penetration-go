<!--
 * @Author: Youwei Li
 * @Date: 2021-12-27 17:22:01
 * @LastEditTime: 2021-12-27 17:41:23
 * @LastEditors: Youwei Li
 * @Description: 
-->
# Intranet-Penetration-Go

## Software function:

It allows the world to access websites on home computers.

## Principle
The client runs on a home computer with its own website. User is the browser that accesses the web site.

## Intranet penetration v1 Version 2 features
1. Multiple processes manage multiple TCP links concurrently. Faster.
2. Add heartbeat packet mechanism (20 seconds). Deal with extreme situations such as unplugging the network cable
3. Customize the listening port of server and home computer
4. Support disconnection and reconnection. If you don't receive a heartbeat packet or the network is disconnected, it will be reconnected automatically
5. The server side basically does not need to be shut down. If it needs to be restarted, only the client can be restarted
6. This version is a refactored version. There is no need to introduce additional packages. Only two files are required. The code is clear and easy to understand, and a large number of explanations are added.

## Usage:
1. Configure the go locale,
2. Put the server Go upload to the public network server. Running example: go run server Go - localport 3002 - remoteport 20012 (as shown in the figure below) localport is the port accessed by the user, and remoteport is the port for communication with the client.
3. Put the client Go is placed on the home computer (there is no public network IP, only port 80 of the home computer can access the local website). Running example go run client.go - host server IP - localport 80 - remoteport 20012 (as shown below) localport port is the port of the home computer website, remoteport is the port for communication with the server, and the settings must be consistent with the server
4. Any browser in the world can access the website in the home computer by accessing the public network IP: 3002. (if the server IP is 1.1.1.1, access 1.1.1:3002) (as shown below)

## Supplement:
1. The author is new to TCP network programming. The software is not perfect. If a bug is found, you are welcome to submit the issue
2. In addition, if you think it is helpful, you can donate to the project and buy me a cup of cafe.