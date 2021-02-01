function setElemDisplay(elemId, displayValue) {
  let elem = document.getElementById(elemId);
  elem.style.display = displayValue
}

function setChevronDirection(elemId, direction) {
  let elem = document.getElementById(elemId);
  elem.className = `chevron ${direction}`
}

function toggleInfoPanel() {
  const ls = localStorage.getItem('infoPanel')
  if (ls == 'show') {
    localStorage.setItem('infoPanel', 'hide')
    setElemDisplay('infoPanel', 'none')
    setChevronDirection('toggleIcon', 'right')
  } else {
    localStorage.setItem('infoPanel', 'show')
    setElemDisplay('infoPanel', 'block')
    setChevronDirection('toggleIcon', 'down')
  }
}

function infoPanelInitPageLoad() {
  const ls = localStorage.getItem('infoPanel')
  if (ls == 'hide') {
    setElemDisplay('infoPanel', 'none')
    setChevronDirection('toggleIcon', 'right')
  } else {
    setElemDisplay('infoPanel', 'block')
    setChevronDirection('toggleIcon', 'down')
    localStorage.setItem('infoPanel', 'show')
  }
}

infoPanelInitPageLoad()
