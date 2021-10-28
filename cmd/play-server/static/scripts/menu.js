function handleMenuToggle() {
  let elements = document.getElementById('menuToggle').childNodes;
  let secondaryNav = document.getElementById('secondaryNav');
  if (elements[1].classList.contains('hidden')) {
    elements[1].classList.remove('hidden');
    elements[3].classList.add('hidden');
    secondaryNav.style.display = "none";
  } else if (elements[3].classList.contains('hidden')) {
    elements[1].classList.add('hidden');
    elements[3].classList.remove('hidden');
    secondaryNav.style.display = "block";
  }
}


function setHomeLink(sessionId) {
  let sessionExitExtension = '';
  if (sessionId !== null) {
    sessionExitExtension = `session-exit?s=${sessionId}`
  }
  if (window.location.origin === 'https://couchbase.live/') {
    localStorage.setItem('homeUrl', 'https://couchbase.live/')
    document.getElementById("homeLink").href = `${window.location.origin}${sessionExitExtension}`
  } else {
    localStorage.setItem('homeUrl', window.location.origin)
    document.getElementById("homeLink").href = `/${sessionExitExtension}`
  }
}

function setHomeLinkSession() {
  console.log(localStorage.getItem('homeUrl'));
  document.getElementById("homeLink").href = localStorage.getItem('homeUrl') !== null ?  localStorage.getItem('homeUrl') : 'https://couchbase.live/';
  console.log(document.getElementById("homeLink").href);
  console.log(document.getElementById("homeLink"));
}

