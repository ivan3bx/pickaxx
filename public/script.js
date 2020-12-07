import * as messageBox from './messageBox.mjs'

var inputForm = null
var inputBox = null
var startButton = null
var stopButton = null

var app = {
    init: () => {
        // DOM fields
        inputForm = document.querySelector('#input-form');
        inputBox = document.querySelector("#input-box");
        startButton = document.getElementById("startButton");
        stopButton = document.getElementById("stopButton");

        // setup initial state
        inputForm.addEventListener('submit', sendCommand);
        startButton.addEventListener('click', startServer);
        stopButton.addEventListener('click', stopServer);

        messageBox.init(startButton, stopButton);
        messageBox.resetScroll();

        // check for WebSocket support (required).
        if (window["WebSocket"] === undefined) {
            console.error("WebSocket support not found.")
            return;
        };
    },
}

document.addEventListener("DOMContentLoaded", app.init);
document.onkeydown = handleKeyDown;

// Handle key-down events.
function handleKeyDown(event) {
    event = event || window.event;

    if (document.activeElement != inputBox && event.key == '/') {
        // '/' will focus & auto-populate input-box
        inputBox.focus();
    }

    if (event.key == 'l' && event.ctrlKey) {
        // ctrl-l will clear the screen
        messageBox.clear();
    }
}

function startServer() {
    ajaxRequest('POST', '/start').send();
}

function stopServer() {
    ajaxRequest('POST', '/stop').send();
}

function sendCommand(event) {
    event.preventDefault();

    if (inputBox.value == "") {
        return;
    }

    let xhr = ajaxRequest('POST', event.currentTarget.action);

    xhr.onload = () => {
        inputBox.value = "";
    }

    xhr.send(JSON.stringify({ "command": inputBox.value }));
}


// Creates a new AJAX POST request
function ajaxRequest(method, url) {
    let xhr = new XMLHttpRequest();
    xhr.open(method, url, true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onload = () => {
        if (xhr.responseText !== "") {
            console.log(xhr.responseText); // debugging
        }
    };

    xhr.onreadystatechange = () => {
        if (xhr.readyState == 4 && xhr.status > 399) {
            console.log(xhr.responseText);
        }
    };

    return xhr;
}
