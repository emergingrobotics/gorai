# GPS Data Processing and Publishing Plan

## Objective
Create a Go-based program that:
1. Reads NMEA GPS data from a serial port configured at 4800 baud, 8N1.
2. Parses the NMEA sentences into structured data.
3. Converts the parsed data into JSON format.
4. Publishes the JSON data to a NATS topic named "gps".

## Prerequisites
- Go installed on your system.
- Access to a serial port with GPS data (e.g., `/dev/ttyUSB0` or `COM3`).
- NATS server running.

## Steps

1. **Set Up Project Structure**
   - Create a new directory for the project.
   - Initialize a Go module.

2. **Install Required Packages**
   - Install packages for serial communication and JSON handling.
   - Install NATS client package.

3. **Implement Serial Port Communication**
   - Open the serial port with the specified configuration (4800 baud, 8N1).
   - Read data from the serial port in a loop.

4. **Parse NMEA Sentences**
   - Use an existing library to parse NMEA sentences into structured data.
   - Handle common NMEA sentences like GPGGA and GPRMC.

5. **Convert Data to JSON**
   - Convert the parsed NMEA data into JSON format.

6. **Publish JSON Data to NATS**
   - Connect to a running NATS server.
   - Publish the JSON data to the "gps" topic.

7. **Error Handling and Logging**
   - Implement error handling for serial port operations, parsing, and NATS publishing.
   - Add logging for debugging and monitoring.

8. **Testing**
   - Test the program with a GPS device connected to the specified serial port.
   - Verify that JSON data is correctly published to the "gps" topic.

## Detailed Implementation

1. **Set Up Project Structure**

   ```sh
   mkdir gorai-gps-example
   cd gorai-gps-example
   go mod init github.com/gorai/gorai-gps-example
   ```

2. **Install Required Packages**

   ```sh
   go get github.com/tarm/serial
   go get github.com/marcellof23/nmea
   go get nats.io/nats.go
   ```

3. **Implement Serial Port Communication**

   Create a file `main.go` and add the following code:

   ```go
   package main

   import (
       "bufio"
       "context"
       "encoding/json"
       "fmt"
       "log"
       "os"
       "time"

       "github.com/marcellof23/nmea"
       "github.com/tarm/serial"
       nats "nats.io/nats.go"
   )

   func main() {
       // Serial port configuration
       config := &serial.Config{
           Name:        "/dev/ttyUSB0", // Change this to your serial port
           Baud:        4800,
           ReadTimeout: time.Second * 5,
       }

       // Open the serial port
       port, err := serial.OpenPort(config)
       if err != nil {
           log.Fatalf("Failed to open serial port: %v", err)
       }
       defer port.Close()

       // Connect to NATS server
       nc, err := nats.Connect(nats.DefaultURL)
       if err != nil {
           log.Fatalf("Failed to connect to NATS: %v", err)
       }
       defer nc.Close()

       // Create a scanner to read from the serial port
       scanner := bufio.NewScanner(port)

       for scanner.Scan() {
           line := scanner.Text()
           sentence, err := nmea.Parse(line)
           if err != nil {
               log.Printf("Failed to parse NMEA sentence: %v", err)
               continue
           }

           // Convert the parsed data to JSON
           jsonData, err := json.Marshal(sentence)
           if err != nil {
               log.Printf("Failed to marshal JSON: %v", err)
               continue
           }

           // Publish the JSON data to the "gps" topic
           if err := nc.Publish("gps", jsonData); err != nil {
               log.Printf("Failed to publish to NATS: %v", err)
               continue
           }
       }

       if err := scanner.Err(); err != nil {
           log.Fatalf("Error reading from serial port: %v", err)
       }
   }
   ```

4. **Parse NMEA Sentences**
   - The `github.com/marcellof23/nmea` package is used to parse NMEA sentences.

5. **Convert Data to JSON**
   - The `encoding/json` package is used to convert the parsed data into JSON format.

6. **Publish JSON Data to NATS**
   - The `nats.io/nats.go` package is used to connect to a NATS server and publish messages.

7. **Error Handling and Logging**
   - Error handling is implemented for serial port operations, parsing, and NATS publishing.
   - Logging is added using the `log` package for debugging and monitoring.

8. **Testing**
   - Connect a GPS device to the specified serial port.
   - Run the program and verify that JSON data is correctly published to the "gps" topic.
