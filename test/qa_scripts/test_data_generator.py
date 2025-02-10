#!/usr/bin/env python3

import random
import time
import subprocess
import sys
import signal

def generate_random_data(size_mb):
    # Generate random data of specified size
    return b''.join([bytes([random.randint(0, 255)]) for _ in range(size_mb * 1024 * 1024)])

def send_data(data, port):
    try:
        # Use netcat to send data through the tunnel
        proc = subprocess.Popen(['nc', 'localhost', str(port)], stdin=subprocess.PIPE)
        proc.communicate(input=data)
    except Exception as e:
        print(f"Error sending data: {e}")

def main():
    if len(sys.argv) != 3:
        print("Usage: test_data_generator.py <tunnel_port> <duration_seconds>")
        sys.exit(1)

    port = int(sys.argv[1])
    duration = int(sys.argv[2])
    start_time = time.time()

    def signal_handler(signum, frame):
        print("\nStopping data generation...")
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)

    print(f"Starting data generation for {duration} seconds...")
    while time.time() - start_time < duration:
        # Generate random size between 10MB and 100MB
        size = random.randint(10, 100)
        print(f"Sending {size}MB of data...")
        
        data = generate_random_data(size)
        send_data(data, port)
        
        # Random delay between 1-5 seconds
        delay = random.uniform(1, 5)
        time.sleep(delay)

    print("Data generation completed")

if __name__ == "__main__":
    main()
