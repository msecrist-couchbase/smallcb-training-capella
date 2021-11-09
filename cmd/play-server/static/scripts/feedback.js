function handleFeedbackForm() {
  document.getElementById('src_url').value = window.location.href;
  const feedbackForm = document.getElementById('feedback');
  feedbackForm.addEventListener('submit', (event) => {
    console.log("testing");
    event.preventDefault();
    const pageWasHelpful = event.target.elements[1].checked;
    const suggestionMsg = event.target.elements[3].value;
    const srcUrl = event.target.elements[4].value;

    // TODO: use feedbackURL flag here instead
    const feedbackUrl = 'https://devportal-api.prod.couchbase.live/pageLikes';

    const data = new URLSearchParams();
    data.append('liked', pageWasHelpful);
    data.append('message', suggestionMsg);
    data.append('src_url', srcUrl);

    fetch(feedbackUrl, {
      method: 'POST',
      body: data
    }).then((result) => {
      displayFormMessage('Thanks for the Feedback!')
    }).catch((error) => {
      console.log("FETCH error: ");
      console.log(error);
      displayFormMessage('An error occurred while submitting feedback.')
    })
  })
}


function displayFormMessage(message) {
  document.querySelector('fieldset.feedback-fields').style.display = 'none'
  document.getElementById('result').innerText = message
}
