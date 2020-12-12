import * as messageBox from './messages.js';

let inputForm = null;
let inputBox = null;
let startButton = null;
let stopButton = null;
let fileDrop = null;

// Creates a new AJAX POST request
function ajaxRequest(method, url) {
  const xhr = new XMLHttpRequest();
  xhr.open(method, url, true);
  xhr.setRequestHeader('Content-Type', 'application/json');

  // xhr.onload = () => {
  //   if (xhr.responseText !== '') {
  //     console.log(xhr.responseText); // debugging
  //   }
  // };

  // xhr.onreadystatechange = () => {
  //   if (xhr.readyState === 4 && xhr.status > 399) {
  //     console.log(xhr.responseText);
  //   }
  // };

  return xhr;
}

function sendCommand(event) {
  event.preventDefault();

  if (inputBox.value === '') {
    return;
  }

  const xhr = ajaxRequest('POST', event.currentTarget.action);

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
    fileDrop = document.querySelector('.drop-zone');

    // setup initial state
    inputForm.addEventListener('submit', sendCommand);
    startButton.addEventListener('click', () => { ajaxRequest('POST', '/start').send(); });
    stopButton.addEventListener('click', () => { ajaxRequest('POST', '/stop').send(); });

    fileDrop.addEventListener('dragenter', (e) => {
      e.preventDefault();
      e.stopPropagation();

      for (let i = 0; i < fileDrop.children.length; i += 1) {
        fileDrop.children[i].classList.add('dragging-in-progress');
      }

      fileDrop.classList.add('highlight');
      return false;
    });

    fileDrop.addEventListener('dragleave', (e) => {
      e.preventDefault();
      e.stopPropagation();

      for (let i = 0; i < fileDrop.children.length; i += 1) {
        fileDrop.children[i].classList.remove('dragging-in-progress');
      }

      fileDrop.classList.remove('highlight');
      return false;
    });

    fileDrop.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.stopPropagation();
      return false;
    });

    fileDrop.addEventListener('drop', (e) => {
      e.preventDefault();
      e.stopPropagation();

      for (let i = 0; i < fileDrop.children.length; i += 1) {
        fileDrop.children[i].classList.remove('dragging-in-progress');
      }
      fileDrop.classList.remove('highlight');

      console.log(`filename:${e.dataTransfer.files.item(0).name}`);
      console.log(`fileSize:${e.dataTransfer.files.item(0).size}`); // bytes (37,961,464)
      console.log(`fileType:${e.dataTransfer.files.item(0).type}`); // should be 'application/java-archive'
    });

    messageBox.init(startButton, stopButton);
    messageBox.resetScroll();

    // check for WebSocket support (required).
    if (window.WebSocket === undefined) {
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
