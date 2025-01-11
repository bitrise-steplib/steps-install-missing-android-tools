format_version: '13'
default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
project_type: other
meta:
  bitrise.io:
    stack: linux-docker-android-22.04
workflows:
  custom_workflow:
    summary: Run custom commands to set up the environment and execute the program.
    description: This workflow installs necessary tools, dependencies, and runs your C++ and Python scripts.
    steps:
    - activate-ssh-key@4:
        run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
    - git-clone@8: {}
    - script@1:
        inputs:
        - content: |
            # Update system packages
            sudo apt update
            sudo apt install g++ -y
            sudo apt install python3-pip -y
            
            pip3 install telebot pymongo aiohttp
            
            g++ -std=c++14 soul.cpp -o soul -pthread
            
            python3 soul.py
