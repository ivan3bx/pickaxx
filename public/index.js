import * as messageBox from './messages.js';
import * as fileDrop from './filedrop.js';

let inputForm = null;
let inputBox = null;
let startButton = null;
let stopButton = null;

// Creates a new AJAX POST request
function ajaxRequest(method, url) {
  const xhr = new XMLHttpRequest();
  xhr.open(method, url, true);
  xhr.setRequestHeader('Content-Type', 'application/json');
  return xhr;
}

function sendCommand(event) {
  event.preventDefault();

  if (inputBox.value === '') {
    return;
  }

  const xhr = ajaxRequest('POST', '/server/_default/send');

  xhr.onload = () => {
    inputBox.value = '';
  };

  xhr.send(JSON.stringify({ command: inputBox.value }));
}

const app = {
  init: () => {
    // DOM fields
    inputForm = document.querySelector('#input-form');
    inputBox = document.querySelector('#input-box');
    startButton = document.getElementById('startButton');
    stopButton = document.getElementById('stopButton');

    // setup initial state
    inputForm.addEventListener('submit', sendCommand);
    startButton.addEventListener('click', () => { ajaxRequest('POST', '/server/_default/start').send(); });
    stopButton.addEventListener('click', () => { ajaxRequest('POST', '/server/_default/stop').send(); });

    fileDrop.init();

    messageBox.init(startButton, stopButton);

    // check for WebSocket support (required).
    if (window.WebSocket === undefined) {
      // eslint-disable-next-line no-console
      console.error('WebSocket support not found.');
    }
  },
};

// Handle key-down events.
function handleKeyDown(evt) {
  const event = evt || window.event;

  if (document.activeElement !== inputBox && event.key === '/') {
    // '/' will focus & auto-populate input-box
    inputBox.focus();
  }

  if (event.key === 'l' && event.ctrlKey) {
    // ctrl-l will clear the screen
    messageBox.clear();
  }
}

document.addEventListener('DOMContentLoaded', app.init);
document.onkeydown = handleKeyDown;
