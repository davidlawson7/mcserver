#!/bin/sh

# Install required files
rpm --import https://yum.corretto.aws/corretto.key
curl -L -o /etc/yum.repos.d/corretto.repo https://yum.corretto.aws/corretto.repo
yum install -y java-17-amazon-corretto-devel.x86_64

# Create Minecraft (MC) user account
adduser minecraft

# Create filesystem for MC
mkdir /opt/minecraft/
mkdir /opt/minecraft/server/
cd /opt/minecraft/server

# TODO: Fetch minecraft from backup

# Change ownership of folder
chown -R minecraft:minecraft /opt/minecraft/

# TODO: Fetch minecraft.service and move to /etc/systemd/system/minecraft.service

# Setup minecraft service
chmod 664 /etc/systemd/system/minecraft.service
systemctl daemon-reload
