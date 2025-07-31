#!/usr/bin/env python3
"""
Simple UDP Client for testing FRP implementation
This client will send messages through the FRP tunnel to reach the server
"""

import socket
import threading
import logging
import time
import signal
import sys
from datetime import datetime

# Configure detailed logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - [%(threadName)s] - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
    ]
)

logger = logging.getLogger(__name__)

class UDPClient:
    def __init__(self, server_host='127.0.0.1', server_port=35561):
        self.server_host = server_host
        self.server_port = server_port
        self.socket = None
        self.connected = False
        self.message_count = 0
        
    def connect(self):
        """Create UDP socket for communication"""
        try:
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            self.socket.settimeout(10.0)  # Set timeout for receive operations
            
            logger.info(f"UDP client ready to communicate with FRP server at {self.server_host}:{self.server_port}")
            logger.info(f"Local address: {self.socket.getsockname()}")
            
            self.connected = True
            return True
            
        except Exception as e:
            logger.error(f"Failed to create UDP socket: {e}")
            return False
            
    def send_message(self, message):
        """Send a message to the server"""
        if not self.connected:
            logger.error("Not connected to server")
            return False
            
        try:
            # Send message
            self.socket.sendto(message.encode('utf-8'), (self.server_host, self.server_port))
            self.message_count += 1
            logger.info(f"Sent message #{self.message_count}: {message}")
            
            # Wait for response
            try:
                response_data, server_addr = self.socket.recvfrom(4096)
                response = response_data.decode('utf-8')
                logger.info(f"Received response from {server_addr}: {response}")
                return True
            except socket.timeout:
                logger.warning("Timeout waiting for response")
                return False
            except Exception as e:
                logger.error(f"Error receiving response: {e}")
                return False
                
        except Exception as e:
            logger.error(f"Failed to send message: {e}")
            return False
            
    def send_test_messages(self, count=5, interval=2.0):
        """Send a series of test messages"""
        logger.info(f"Starting automated test: {count} messages with {interval}s interval")
        
        successful_sends = 0
        for i in range(count):
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            message = f"Test message {i+1}/{count} at {timestamp}"
            
            if self.send_message(message):
                successful_sends += 1
            else:
                logger.error(f"Failed to send message {i+1}")
                
            if i < count - 1:  # Don't sleep after the last message
                time.sleep(interval)
                
        logger.info(f"Test completed: {successful_sends}/{count} messages sent successfully")
        return successful_sends == count
        
    def interactive_mode(self):
        """Interactive mode for manual testing"""
        logger.info("Entering interactive mode. Type 'quit' to exit.")
        
        while True:
            try:
                message = input("Enter message: ").strip()
                if message.lower() in ['quit', 'exit', 'q']:
                    break
                    
                if message:
                    self.send_message(message)
                    
            except EOFError:
                break
            except KeyboardInterrupt:
                logger.info("Keyboard interrupt in interactive mode")
                break
                
    def disconnect(self):
        """Close the connection"""
        if self.socket:
            self.socket.close()
            self.connected = False
            logger.info("Disconnected from server")
            logger.info(f"Total messages sent: {self.message_count}")

def run_performance_test(client, duration=30):
    """Run a performance test"""
    logger.info(f"Starting performance test for {duration} seconds")
    
    start_time = time.time()
    message_count = 0
    successful_sends = 0
    
    while time.time() - start_time < duration:
        message = f"Performance test message {message_count + 1}"
        if client.send_message(message):
            successful_sends += 1
        message_count += 1
        time.sleep(0.1)  # Small delay to avoid overwhelming
        
    elapsed_time = time.time() - start_time
    logger.info(f"Performance test completed:")
    logger.info(f"Duration: {elapsed_time:.2f} seconds")
    logger.info(f"Total messages: {message_count}")
    logger.info(f"Successful sends: {successful_sends}")
    logger.info(f"Success rate: {(successful_sends/message_count)*100:.1f}%")
    logger.info(f"Messages per second: {message_count/elapsed_time:.2f}")

if __name__ == "__main__":
    # Parse command line arguments for different test modes
    import argparse
    
    parser = argparse.ArgumentParser(description='UDP Client for FRP testing')
    parser.add_argument('--host', default='10.26.116.54', help='FRP server host')
    parser.add_argument('--port', type=int, default=35561, help='FRP server port')
    parser.add_argument('--mode', choices=['auto', 'interactive', 'performance'], 
                       default='auto', help='Test mode')
    parser.add_argument('--count', type=int, default=5, help='Number of test messages (auto mode)')
    parser.add_argument('--interval', type=float, default=2.0, help='Interval between messages (auto mode)')
    parser.add_argument('--duration', type=int, default=30, help='Duration for performance test (seconds)')
    
    args = parser.parse_args()
    
    # Create and connect client
    client = UDPClient(args.host, args.port)
    
    try:
        logger.info(f"Starting UDP test client for FRP testing (mode: {args.mode})")
        
        if not client.connect():
            logger.error("Failed to create UDP socket, exiting")
            sys.exit(1)
        
        if args.mode == 'auto':
            client.send_test_messages(args.count, args.interval)
        elif args.mode == 'interactive':
            client.interactive_mode()
        elif args.mode == 'performance':
            run_performance_test(client, args.duration)
            
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
    finally:
        client.disconnect()
        logger.info("UDP client stopped")
