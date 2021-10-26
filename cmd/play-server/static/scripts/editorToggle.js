function toggleEditorView() {
  let isSideBySide = document.getElementById("editorWrapper").classList.contains('row');
  let iframe = document.getElementById("code-output").childNodes[1].querySelector("iframe");
  let innerDoc = iframe.contentDocument || iframe.contentWindow.document;

  getOutputHeight()

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
  }

}

// Helper to adjust the main info and editor columns for side by side mode
function adjustOuterColumns(isSideBySide) {
  if (isSideBySide) {
    // to VERTICAL STACK view
    document.getElementById("editorColumn").classList.remove('col-md-8')
    document.getElementById("editorColumn").classList.add('col-md-7')
    document.getElementById("infoColumn").classList.remove('col-md-4')
    document.getElementById("infoColumn").classList.add('col-md-5')
  } else {
    // to SIDE BY SIDE view
    document.getElementById("editorColumn").classList.add('col-md-8')
    document.getElementById("editorColumn").classList.remove('col-md-7')
    document.getElementById("infoColumn").classList.add('col-md-4')
    document.getElementById("infoColumn").classList.remove('col-md-5')
  }
}

function getOutputHeight() {
  let iframe = document.getElementById("code-output").childNodes[1].querySelector("iframe");
  let innerDoc = iframe.contentDocument || iframe.contentWindow.document;

  if (innerDoc.getElementsByTagName("body")[0].childNodes.length === 3 && innerDoc.getElementsByTagName("body")[0].childNodes[1].tagName === 'PRE') {
    let updatedHeight = Math.max(innerDoc.getElementsByTagName("body")[0].childNodes[1].offsetHeight + 24, 488)
    iframe.style.height = `${updatedHeight}px`
  }
}
