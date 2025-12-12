# Gorai

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

This project aims to be an alternative to ROS2 or YARP.  We aim to have the following attributes:

- be written in Go (and when appropriate, tinygo)
- use AI assisted coding whereever possible
- use NATS.io extensively 
- define appropriate data types for robotics
- enable use of TPU/NPU in robotics to the maximum extent possible
- have a low barrier to entry for adoptions
- be as modular as possible
- take the lessons from distributed cloud software and apply it
- HAVE FUN!

We aim to become the robotics platform of choice for:

- people interested in building new, modern robotics
- who do not need interoperability with other platforms
- who are open to experimentation
- who want more than python but not the complexity of C++
- who value extensability and performance
- who want to use AI assisted development
- who want the more powerful parts of the robot to be Linux-based
- who want to use Go, and for microcontrollers, use tinygo

## Why Gorai

- Gorai learns from three generations of robotics middleware and the entire cloud distributed system experience
- Gorai has a "Robotics Definition Language" (RDL) that defines software not physical attributes
- Gorai is co-designed by AI with AI development as a prime goal
- Gorai auto-generates most of the plumbing code for you, based on the RDL and using AI
- Gorai is intended from the start to use AI/ML inferencing on the robot
- Gorai assumes actual Linux devices as the core brain - think Raspberry Pi 5
- Gorai uses core building blocks proven in the real world - NATS.io and Prometheus
- Gorai is commercially-friendly open (source, 3D printing, hardware)
- Gorai wants to democratize the creation and use of prosumer robotics 
- This means making "real" robots (not toys) buildable by the experienced maker
