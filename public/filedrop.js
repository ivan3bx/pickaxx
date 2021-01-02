// jQuery dependency for modals..
const { $ } = window;

const dropZone = document.querySelector('.drop-zone');
const progressBar = document.querySelector('div[role=progressbar]');
const newServerModal = document.querySelector('#new-server-modal');
const saveButton = document.querySelector('#new-server-modal .btn-primary');
const cancelButton = document.querySelector('#new-server-modal .btn-secondary');
const formEl = newServerModal.querySelector('form');
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

  // see https://code-boxx.com/simple-drag-and-drop-file-upload/
  const data = new FormData();
  data.append('file', file);

  const xhr = new XMLHttpRequest();
  xhr.open('POST', '/server');
  saveButton.disabled = true;

  // XHR - on successful load
  xhr.onload = () => {
    if (xhr.readyState !== xhr.DONE) { return; }

    if (xhr.status > 299) {
      console.log(`Error: ${xhr.responseText}`);
      return;
    }

    // success!
    const xhrRsp = JSON.parse(xhr.responseText);

    // set placeholder name
    serverNameField.value = xhrRsp.name;

    // set button state
    saveButton.disabled = false;
    newServerModal.dataset.key = xhrRsp.key; // used to commit change

    setTimeout(() => {
      progressBar.classList.add('bg-success');
      progressBar.classList.remove('progress-bar-animated', 'progress-bar-striped');
    }, 800);

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
// dropCancel - user cancels creation of a new server
//
function dropCancel(e) {
  const { key } = newServerModal.dataset;

  if (key === 'undefined') {
    return false;
  }

  const data = new FormData();
  data.append('key', key);

  const xhr = new XMLHttpRequest();

  // fire & forget
  xhr.open('DELETE', '/server');
  xhr.send(data);
}

//
// dropCommit - user commits creation of new server
//
function dropCommit(e) {
  const { key } = newServerModal.dataset;

  const data = new FormData();
  data.append('key', key);
  data.append('name', serverNameField.value);

  const xhr = new XMLHttpRequest();
  xhr.open('PUT', '/server');
  saveButton.disabled = true;

  // XHR - on successful load
  xhr.onload = () => {
    if (xhr.readyState !== xhr.DONE) { return; }

    if (xhr.status > 299) {
      dropZone.innerHTML += `<div>Error: ${xhr.responseText}</div>`;
      return;
    }

    // success! redirect to new server
    const xhrRsp = JSON.parse(xhr.responseText);
    window.location.href = xhrRsp.url;
  };

  xhr.send(data);
}

export function init() {
  dropZone.addEventListener('dragenter', dropEnter);
  dropZone.addEventListener('dragleave', dropLeave);
  dropZone.addEventListener('dragover', dropOver);
  dropZone.addEventListener('drop', dropStage);

  formEl.addEventListener('submit', dropCommit);
  saveButton.addEventListener('click', dropCommit);
  cancelButton.addEventListener('click', dropCancel);

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
