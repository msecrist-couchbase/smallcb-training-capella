function setElemDisplay(elemId, displayValue) {
  let elem = document.getElementById(elemId);
  elem.style.display = displayValue
}

function toggleInfoPanel() {
  const ls = localStorage.getItem('infoPanel')
  if (ls === 'show') {
    localStorage.setItem('infoPanel', 'hide')
    setElemDisplay('infoPanel', 'none')
  } else if (ls === 'hide') {
    localStorage.setItem('infoPanel', 'show')
    setElemDisplay('infoPanel', 'block')
  }
}

function infoPanelInitPageLoad() {
  const ls = localStorage.getItem('infoPanel')
  if(ls == 'hide') {
    setElemDisplay('infoPanel', 'none')
  } else if(ls == 'show') {
    setElemDisplay('infoPanel', 'block')
  } else if(ls === null) {
    localStorage.setItem('infoPanel', 'show')
  }
}

infoPanelInitPageLoad()