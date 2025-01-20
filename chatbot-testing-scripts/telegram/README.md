# Telegram Chatbot E2EE and Hide Sender Analysis

This repository contains the script referenced in Section 4.2, "Platforms with Chatbot Support", of our research. The script is designed to construct a basic chatbot. Additionally, the HTTP messages generated during the script's execution have been collected and are included.

### 1. `index.js`
A JavaScript-based script designed to poll the Telegram server via HTTP. It retrieves the latest messages and responds to them accordingly. The script provides foundational automation for interacting with the Telegram platform, leveraging polling techniques to ensure consistent communication.

### 2. `privacy-mode.http.txt` and `non-privacy-mode.http.txt`
These files contain HTTP messages intercepted during communication with the Telegram server. The data was collected using Wireshark and provides evidence of the following observations in Section 4.2:

- **Group E2EE with chatbots**: Messages are transmitted securely via TLS; however, the content remains unencrypted beyond transport-level protection.
- **Hide sender**: Chatbots are capable of accessing the sender's User ID along with the message content.

## Usage  

### Installation  
To replicate our analysis, ensure the following software dependencies are met:  

1. **Node.js**  
   Install the Node.js runtime from the official website: [Node.js — Download Node.js®](https://nodejs.org/en/download).  
   - **Version Used:** v20.18.0 (for consistency with this analysis).  

2. **Repository Setup**  
   Navigate to the root directory of the repository and execute the following command to install the necessary dependencies:  
   ```bash  
   npm install  
   ```  

3. **Wireshark**  
   Install Wireshark from [Wireshark · Download](https://www.wireshark.org/download.html).  

### Launch Chatbot  

Follow these steps to execute the chatbot and capture network traffic:  

1. **Start Wireshark**  
   Launch the Wireshark GUI.
2. **Set Up TLS Key Logging**  
   Specify the `(Pre)-Master-Secret log filename` in Wireshark’s preferences, as outlined in the official documentation: [TLS - Wireshark Wiki](https://wiki.wireshark.org/TLS#preference-settings).  
3. **Start Packet Capture**  
   Select the appropriate network interface for internet traffic and begin capturing packets.
4. **Run the Chatbot**  
   In the root directory of this repository, execute the chatbot script with the following command:  
   ```bash  
   node --tls-keylog="/tmp/keylog.txt" index.js  
   ```  
   - Ensure the `--tls-keylog` flag points to the same filename specified in Step 2.  
   - For more details on the `--tls-keylog` flag, refer to the Node.js documentation: [Command-line API | Node.js v20.18.0 Documentation](https://nodejs.org/download/release/v20.18.0/docs/api/cli.html#--tls-keylogfile).  
5. **Interact with the Chatbot**  
   Send a message within a Telegram group to trigger chatbot functionality.  
6. **Filter Network Traffic**  
   In the Wireshark interface, apply the following filter to isolate HTTP messages:  
   ```text  
   _ws.col.protocol == "HTTP"  
   ```  
