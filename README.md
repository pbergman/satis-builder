## Satis Builder

github webhook listener that will (re)build satis in a docker when events are received.

#### dependencies

docker (only server, this app uses the client SDK)

#### Signals

the application will listen to signals which can be used to update or pull image from remote.  

| Signal |Description
|--------|-----
| SIGUSR1 |rebuild satis build
| SIGUSR2 |pull (update) image from docker remote    
    

#### config

The config is based on yaml with the following properties: 


```
# The address to listen on for incoming github events
# default: :8080
#
# listen: <string> 

# The secret that can be configured for signing/validating 
# the github events.
#
# see: https://docs.github.com/en/developers/webhooks-and-events/securing-your-webhooks
# 
# secret: <string>

# The user to run the container, defauls to user that runs this container
#
# user: <string>

# The repositories to manage and when resieving an push event from on of 
# those repositories it will rebuid satis
#
repositories: <list>

# The default satis config in an yaml structure 
#
# see: https://composer.github.io/satis/config
#
satis_config: 

# Docker container specific config 
#
# container:
    # the name of the image, can be used to set an specific version  
    # default: composer/satis
    #
    # name:
    
    # auto remove image when finished
    # default: true
    #
    # remove:
    
    # log driver to read back container output   
    # default: syslog
    #
    # see: https://docs.docker.com/config/containers/logging/configure/
    #
    # log-type:
    
    # log driver options
    #
    # log-args:

# Directories for binding in the conatiner
# 
directories:
    # the ssh directory which has keys that has access to you repository
    ssh:
    
    # composer directory for persistent cache
    #
    # composer:  
    
    # build directory which will be used to dump the satis config and
    # in {build}/out will contain the generated satis files
    # default: {cwd}/build
    #
    build:
```

