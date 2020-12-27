// jQuery dependency for modals..
import $ from 'jquery';

const dropZone = document.querySelector('.drop-zone');
const progressBar = document.querySelector('div[role=progressbar]');
const newServerModal = document.querySelector('#new-server-modal');
const saveButton = document.querySelector('#new-server-modal .btn-primary');
const serverNameField = document.querySelector('#server-name');

//
// dropEnter - fires when user drags into the drop zone.
//
function dropEnter(e) {
  e.preventDefault();
  e.stopPropagation();

  for (let i = 0; i < dropZone.children.length; i += 1) {
    dropZone.children[i].classList.add('dragging-in-progress');
  }

  dropZone.classList.remove('alert-box');
  dropZone.classList.add('highlight');
  return false;
}

//
// dropLeave - fires when user drags out of the drop zone.
//
function dropLeave(e) {
  e.preventDefault();
  e.stopPropagation();

  // children should ignore pointer movements..
  for (let i = 0; i < dropZone.children.length; i += 1) {
    dropZone.children[i].classList.remove('dragging-in-progress');
  }

  dropZone.classList.remove('highlight');
  return false;
}

//
// dropOver - fires during user's' hovering inside drop zone.
//
function dropOver(e) {
  e.preventDefault();
  e.stopPropagation();
  return false;
}

//
// dropStage - fires when user releases file into the drop zone. Stages file on server.
//
function dropStage(e) {
  e.preventDefault();
  e.stopPropagation();

  // release children from ignoring pointer movements..
  for (let i = 0; i < dropZone.children.length; i += 1) {
    dropZone.children[i].classList.remove('dragging-in-progress');
  }

  dropZone.classList.remove('highlight');

  const file = e.dataTransfer.files.item(0);

  // check mime type
  if (file.type !== 'application/java-archive') {
    dropZone.classList.add('alert-box');
    return false;
  }

  // check minimum file size
  if (file.size < 20000000) {
    dropZone.classList.add('alert-box');
    return false;
  }

  $(newServerModal).modal();

  //
  // upload to server!
  //

  // see https://code-boxx.com/simple-drag-and-drop-file-upload/

  const data = new FormData();
  data.append('file', file);

  const xhr = new XMLHttpRequest();
  xhr.open('POST', '/server');
  saveButton.disabled = true;

  // XHR - on successful load
  xhr.onload = () => {
    if (xhr.readyState === xhr.DONE && (xhr.status > 299)) {
      console.log(`Error: ${xhr.responseText}`);
      return;
    }

    // success!
    const xhrRsp = JSON.parse(xhr.responseText);

    dropZone.innerHTML += `<div>${xhr.responseText}</div>`;
    serverNameField.value = xhrRsp.key;
    saveButton.disabled = false;
    saveButton.dataset.key = xhrRsp.key;

    console.log(xhr.responseText);
  };

  // XHR - on progress event
  xhr.upload.addEventListener('progress', (progEvent) => {
    const width = `${Math.ceil((progEvent.loaded / progEvent.total) * 100)}%`;
    progressBar.style.width = width;
  }, false);

  // XHR - on completion or http status code as error
  xhr.onreadystatechange = () => {
    if (xhr.readyState === XMLHttpRequest.DONE) {
      const { status } = xhr;
      if (status > 399) {
        progressBar.classList.remove('progress-bar-animated', 'progress-bar-striped');
        progressBar.classList.add('bg-danger');
        progressBar.appendChild(document.createTextNode('Error: File was not received.'));
        console.log(xhr.responseText);
      }
    }
  };

  xhr.send(data);
  return true;
}

//
// dropCommit - user commits creation of new server.
function dropCommit(e) {
  const { key } = saveButton.dataset;
  console.log("TODO: call 'commit' with key:" + key);
}

export function init() {
  dropZone.addEventListener('dragenter', dropEnter);
  dropZone.addEventListener('dragleave', dropLeave);
  dropZone.addEventListener('dragover', dropOver);
  dropZone.addEventListener('drop', dropStage);

  saveButton.addEventListener('click', dropCommit);

  // set focus for the server modal
  $(newServerModal).on('shown.bs.modal', () => {
    $(serverNameField).trigger('focus');
  });

  $(newServerModal).on('hide.bs.modal', () => {
    progressBar.classList.add('progress-bar-animated', 'progress-bar-striped');
    progressBar.classList.remove('bg-danger');
  });

}

export { init as default };
