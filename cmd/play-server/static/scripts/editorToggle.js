function setEditorView() {
  setOutputHeight()
  // check local storage, if no local storage default to 3 col
  let savedEditorMode = localStorage.getItem('editorMode')

  let isSideBySide = document.getElementById("editorWrapper").classList.contains('row');
  let isSmallScreen = window.innerWidth < 991;



  if (isSmallScreen && isSideBySide) {
    toggleEditorView()
  } else if (savedEditorMode === "stacked" && isSideBySide) {
    toggleEditorView()
  }
}

function toggleEditorView() {
  let isSideBySide = document.getElementById("editorWrapper").classList.contains('row');
  let iframe = document.getElementById("code-output").childNodes[1].querySelector("iframe");
  let innerDoc = iframe.contentDocument || iframe.contentWindow.document;

  updateOutputHeightOnToggle(document.getElementById("code-ace").style.height, isSideBySide)

  adjustOuterColumns(isSideBySide);
  if (isSideBySide) {
    // swap to VERTICAL STACK editor/output
    document.getElementById("editorWrapper").classList.remove('row')
    document.getElementById("editorDiv").classList.remove('col-md-6')
    document.getElementById("code-output").classList.remove('col-md-6')

    // check if any output is rendered before hiding the output on swap
    if (innerDoc.getElementsByTagName("body")[0].childNodes.length === 0) {
      document.getElementById("code-output").classList.add('hidden')
    }
    document.getElementById("viewToggleButton")._tippy.setContent('Switch code editor view to 3 column')

    // swap out toggle button icon
    document.getElementById("viewToggleButton").classList.add('col-btn')
    document.getElementById("viewToggleButton").classList.remove('stacked-btn')

    localStorage.setItem('editorMode', 'stacked')
  } else {
    // swap to SIDE BY SIDE editor/output
    document.getElementById("editorWrapper").classList.add('row')
    document.getElementById("editorDiv").classList.add('col-md-6')
    document.getElementById("code-output").classList.add('col-md-6')

    if (document.getElementById("code-output").classList.contains('hidden')) {
      document.getElementById("code-output").classList.remove('hidden')
    }
    document.getElementById("viewToggleButton")._tippy.setContent('Switch code editor view to vertical stack')

    // swap out toggle button icon
    document.getElementById("viewToggleButton").classList.add('stacked-btn')
    document.getElementById("viewToggleButton").classList.remove('col-btn')

    localStorage.setItem('editorMode', 'sideBySide')
  }

}

// Helper to adjust the main info and editor columns for side by side mode
function adjustOuterColumns(isSideBySide) {
  if (isSideBySide) {
    // to VERTICAL STACK view
    document.getElementById("editorColumn").classList.remove('col-md-9')
    document.getElementById("editorColumn").classList.add('col-md-7')
    document.getElementById("infoColumn").classList.remove('col-md-3')
    document.getElementById("infoColumn").classList.add('col-md-5')
  } else {
    // to SIDE BY SIDE view
    document.getElementById("editorColumn").classList.add('col-md-9')
    document.getElementById("editorColumn").classList.remove('col-md-7')
    document.getElementById("infoColumn").classList.add('col-md-3')
    document.getElementById("infoColumn").classList.remove('col-md-5')
  }
}

function setOutputHeight() {
  let editorHeight = document.getElementById("code-ace").style.height;
  let isSideBySide = document.getElementById("editorWrapper").classList.contains('row');

  let iframe = document.getElementById("code-output").childNodes[1].querySelector("iframe");
  let innerDoc = iframe.contentDocument || iframe.contentWindow.document;

  if (isSideBySide) {
    iframe.style.height = `${parseInt(editorHeight, 10) - 16}px`
    enableRunButton();
  } else {
    if (innerDoc.getElementsByTagName("body")[0].childNodes.length === 3 && innerDoc.getElementsByTagName("body")[0].childNodes[1].tagName === 'PRE') {
      enableRunButton();
      let updatedHeight = innerDoc.getElementsByTagName("body")[0].childNodes[1].offsetHeight + 28;
      iframe.style.height = `${updatedHeight}px`
    } else {
      iframe.style.height = `350px`
    }
  }
}

function updateRunButtonState() {
  disableRunButton();
  enableRunButtonAfterTimeout(50000);
}

function disableRunButton() {
  if (!document.getElementById("run").classList.contains("disabled")) {
    document.getElementById("run").classList.add("disabled");
  }
}

function enableRunButton() {
  if (document.getElementById("run").classList.contains("disabled")) {
    document.getElementById("run").classList.remove("disabled");
  }
}

function enableRunButtonAfterTimeout(timeout) {
  setTimeout(() => {
    enableRunButton();
  }, timeout)
}

function updateOutputHeightOnToggle(editorHeight, isSideBySide) {
  let iframe = document.getElementById("code-output").childNodes[1].querySelector("iframe");
  let innerDoc = iframe.contentDocument || iframe.contentWindow.document;

  if (!isSideBySide) {
    iframe.style.height = `${parseInt(editorHeight, 10) - 16}px`
  } else {
    if (innerDoc.getElementsByTagName("body")[0].childNodes.length === 3 && innerDoc.getElementsByTagName("body")[0].childNodes[1].tagName === 'PRE') {
      let updatedHeight = innerDoc.getElementsByTagName("body")[0].childNodes[1].offsetHeight + 28;
      iframe.style.height = `${updatedHeight}px`
    } else {
      iframe.style.height = `350px`
    }
  }
}
