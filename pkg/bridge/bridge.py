import sys
import os
import time
import subprocess
import signal
import argparse
import platform

process = None

def is_parent_alive(pid):
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False

def kill_child():
    global process
    if process:
        try:
            if platform.system() != "Windows":
                try:
                    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
                except:
                    process.terminate()
            else:
                process.terminate()
            
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                 process.kill()
        except:
             pass

def signal_handler(signum, frame):
    kill_child()
    sys.exit(0)

def main():
    global process
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    parser = argparse.ArgumentParser(description='Process Bridge')
    parser.add_argument('--parent-pid', type=int, required=True, help='Parent PID to monitor')
    parser.add_argument('--cmd', nargs='+', required=True, help='Command to run')
    args = parser.parse_args()

    parent_pid = args.parent_pid
    cmd = args.cmd

    # Start the child process
    kwargs = {}
    if platform.system() != "Windows":
        kwargs['start_new_session'] = True
    
    try:
        process = subprocess.Popen(
            cmd,
            stdout=sys.stdout,
            stderr=sys.stderr,
            stdin=sys.stdin,
            **kwargs
        )
    except Exception as e:
        print(f"Failed to start process: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        while True:
            # Check if child is still running
            if process.poll() is not None:
                sys.exit(process.returncode)

            # Check if parent is still running
            if not is_parent_alive(parent_pid):
                kill_child()
                sys.exit(0)

            time.sleep(1)

    except Exception:
        kill_child()
        sys.exit(1)

if __name__ == "__main__":
    main()
