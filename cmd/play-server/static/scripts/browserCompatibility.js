function disableSafariButtons() {
  let isIos = /iP(ad|od|hone)/i.test(window.navigator.userAgent);
  let isSafari = /constructor/i.test(window.HTMLElement) || (function (p) { return p.toString() === "[object SafariRemoteNotification]"; })(!window['safari'] || (typeof safari !== 'undefined' && window['safari'].pushNotification));
  if (isSafari) {
    let toDisable = document.getElementsByClassName("disabled-safari");
    for (let i = 0; i < toDisable.length; i++) {
      toDisable[i].classList.add("disabled-tooltip")
      if (toDisable[i].hasAttribute('href')) {
        toDisable[i].removeAttribute('href')
      }
    }

    tippy('.disabled-tooltip', {
      content: 'This feature is not currently supported in Safari. Please use a different browser for access.',
      placement: 'top-start',
      theme: 'custom'
    });
  }

  if (isIos) {
    let toDisable = document.getElementsByClassName("disabled-ios");
    for (let i = 0; i < toDisable.length; i++) {
      toDisable[i].classList.add("disabled")
    }

    let toHide = document.getElementsByClassName("hidden-ios")
    for (let i = 0; i < toHide.length; i++) {
      toHide[i].classList.add('hidden')
      toHide[i].classList.add('disabled')
    }
  }
}
