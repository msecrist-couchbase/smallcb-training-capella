function setElemDisplay(elemId, displayValue) {
  let elem = document.getElementById(elemId);
  elem.style.display = displayValue
}

function setChevronDirection(elemId, direction) {
  let elem = document.getElementById(elemId);
  elem.className = 'chevron'
  elem.className += ` ${direction}`
}

function toggleInfoPanel() {
  const ls = localStorage.getItem('infoPanel')
  if (ls === 'show') {
    localStorage.setItem('infoPanel', 'hide')
    setElemDisplay('infoPanel', 'none')
    setChevronDirection('toggleIcon','down')
  } else if (ls === 'hide') {
    localStorage.setItem('infoPanel', 'show')
    setElemDisplay('infoPanel', 'block')
    setChevronDirection('toggleIcon','up')
  }
}

function infoPanelInitPageLoad() {
  const ls = localStorage.getItem('infoPanel')
  if(ls == 'hide') {
    setElemDisplay('infoPanel', 'none')
    setChevronDirection('toggleIcon','down')
  } else if(ls == 'show') {
    setElemDisplay('infoPanel', 'block')
    setChevronDirection('toggleIcon','up')
  } else if(ls === null) {
    localStorage.setItem('infoPanel', 'show')
  }
}

infoPanelInitPageLoad()