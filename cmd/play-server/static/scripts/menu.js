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
