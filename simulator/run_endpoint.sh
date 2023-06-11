#!/bin/bash

# Set up the routing needed for the simulation
/setup.sh

# The following variables are available for use:
# - ROLE contains the role of this execution context, client or server
# - SERVER_PARAMS contains user-supplied command line parameters
# - CLIENT_PARAMS contains user-supplied command line parameters

# echo "193.167.0.100 client" >> /etc/hosts
# echo "193.167.100.100 server" >> /etc/hosts

if [ "$ROLE" == "client" ]; then
    echo "Wait for the simulator to start up."
    /wait-for-it.sh sim:57832 -s -t 30
    /wait-for-it.sh server:3443 -s -t 5

    echo "Request server HTTP2 to 193.167.100.100:3443"
    echo "Client params: $CLIENT_PARAMS"

    # ./client/client -server=193.167.100.100:4433 $CLIENT_PARAMS
    # ./client/client

    cd client/
    ./client --server=193.167.100.100:3443 $CLIENT_PARAMS
elif [ "$ROLE" == "server" ]; then
    # It is recommended to increase the maximum buffer size (https://github.com/quic-go/quic-go/wiki/UDP-Receive-Buffer-Size)
    # sysctl -w net.core.rmem_max=2500000

    echo "Run the server HTTP2 on 0.0.0.0:3443"
    echo "Server params: $SERVER_PARAMS"

    # ./server/server -addr=0.0.0.0:4433 $SERVER_PARAMS
    # ./server

    cd server/
    ./server --addr=0.0.0.0:3443 $SERVER_PARAMS
fi