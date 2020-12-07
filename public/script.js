document.addEventListener("DOMContentLoaded", () => {
    let messages = document.querySelector(".messages");
    messages.scrollTop = messages.scrollHeight;

    // Handle input-box submission
    document.querySelector('#input-form').addEventListener('submit', (event) => {
        event.preventDefault();
        let inputBox = document.querySelector('#input-box');

        if (inputBox.value == "") {
            return;
        }

        let xhr = ajaxRequest('POST', event.currentTarget.action);
        xhr.onload = () => {
            inputBox.value = "";
        }
        xhr.send(JSON.stringify({ "command": inputBox.value }));
    });

    // handle keyboard-shortcuts
    document.onkeydown = (evt) => {
        let inputBox = document.querySelector("#input-box");
        evt = evt || window.event;
        // '/' will focus & auto-populate input-box
        if (document.activeElement != inputBox && evt.key == '/') {
            inputBox.focus();
        }
        if (evt.key == 'l' && evt.ctrlKey) {
            removeAllChildNodes(document.querySelector(".message-list"));
        }
    };

    // handle websocket connection
    initializeWebSockets();
});

function startServer() {
    ajaxRequest('POST', '/start').send();
};

// Creates a new AJAX POST request
function ajaxRequest(method, url) {
    let xhr = new XMLHttpRequest();
    xhr.open(method, url, true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    // debugging
    xhr.onload = () => {
        console.log(this.responseText);
    };

    xhr.onreadystatechange = () => {
        if (this.readyState == 4 && this.status > 399) {
            console.log(this.responseText);
        }
    };

    return xhr;
}

function stopServer() {
    ajaxRequest('POST', '/stop').send();
};

function logError(errText) {
    let item = document.querySelector(".message-list");
    item.innerHTML = item.innerHTML + "<li>Error: " + (errText || 'Network request failed') + "</li>";
    item.scrollTop = item.scrollHeight;
}

// Websocket handling. Two types of responses are returned:
//
// 1. Server output:
//      { "output" : "text that will appear in the messages-list" }
//
// 2. Process status changes:
//      { "status" : "Starting | Stopping | etc.." }
//
function initializeWebSockets() {
    if (window["WebSocket"]) {
        let startButton = document.getElementById("startButton");
        let stopButton = document.getElementById("stopButton");
        let url = "ws://" + document.location.host + "/ws";

        conn = new ReconnectingWebSocket(url, null, { debug: true, reconnectInterval: 400 });

        conn.onclose = (event) => {
            console.log("wss: close event - " + JSON.stringify(event));
            // Are we running or not? Unclear.
            startButton.disabled = false;
            stopButton.disabled = false;
        }

        conn.onmessage = (event) => {
            let data = JSON.parse(event.data)

            if (data.status !== undefined) {
                if (data.status == "Starting" || data.status == "Running") {
                    startButton.disabled = true;
                    stopButton.disabled = false;
                } else if (data.status == "Stopping") {
                    startButton.disabled = true;
                    stopButton.disabled = true;
                } else {
                    startButton.disabled = false;
                    stopButton.disabled = true;
                }
            } else if (data.output !== undefined) {
                let item = document.querySelector(".message-list");
                let li = document.createElement("li");

                li.appendChild(document.createTextNode(data.output));
                item.appendChild(li);

                item = document.querySelector(".messages")
                item.scrollTop = item.scrollHeight;
            } else {
                console.log("event undefined:" + JSON.stringify(event));
            }
        }
    }
}

function removeAllChildNodes(parent) {
    while (parent.firstChild) {
        parent.removeChild(parent.firstChild);
    }
}
