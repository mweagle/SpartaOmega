#!/bin/bash -xe
SPARTA_OMEGA_BINARY_PATH=/home/ubuntu/{{ .ServiceName }}.lambda.amd64

################################################################################
# 
# Tested on Ubuntu 16.04
#
# AMI: ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20160516.1 (ami-06b94666)
if [ ! -f "/home/ubuntu/userdata.sh" ]
then
  curl -vs http://169.254.169.254/latest/user-data -o /home/ubuntu/userdata.sh
  chmod +x /home/ubuntu/userdata.sh
  apt-get install supervisor -y
fi

# Install everything
service supervisor stop || apt-get install supervisor -y
apt-get update -y 
apt-get upgrade -y 
apt-get install supervisor awscli unzip git -y

################################################################################
# Our own binary
aws s3 cp s3://{{ .S3Bucket }}/{{ .S3Key }} /home/ubuntu/application.zip
unzip -o /home/ubuntu/application.zip -d /home/ubuntu
chmod +x $SPARTA_OMEGA_BINARY_PATH

################################################################################
# SUPERVISOR
# REF: http://supervisord.org/
# Cleanout secondary directory
mkdir -pv /etc/supervisor/conf.d
  
SPARTA_OMEGA_SUPERVISOR_CONF="[program:spartaomega]
command=$SPARTA_OMEGA_BINARY_PATH httpServer
numprocs=1
directory=/tmp
priority=999
autostart=true
autorestart=unexpected
startsecs=10
startretries=3
exitcodes=0,2
stopsignal=TERM
stopwaitsecs=10
stopasgroup=false
killasgroup=false
user=ubuntu
stdout_logfile=/var/log/spartaomega.log
stdout_logfile_maxbytes=1MB
stdout_logfile_backups=10
stdout_capture_maxbytes=1MB
stdout_events_enabled=false
redirect_stderr=false
stderr_logfile=spartaomega.err.log
stderr_logfile_maxbytes=1MB
stderr_logfile_backups=10
stderr_capture_maxbytes=1MB
stderr_events_enabled=false
"
echo "$SPARTA_OMEGA_SUPERVISOR_CONF" > /etc/supervisor/conf.d/spartaomega.conf

# Patch up the directory
chown -R ubuntu:ubuntu /home/ubuntu

# Startup Supervisor
service supervisor restart || service supervisor start