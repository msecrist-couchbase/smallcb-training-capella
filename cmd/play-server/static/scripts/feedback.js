function handleFeedbackForm(feedbackUrl) {
  document.getElementById('src_url').value = window.location.href;
  const feedbackForm = document.getElementById('feedback');

  feedbackForm.addEventListener('submit', (event) => {
    event.preventDefault();
    const pageWasHelpful = event.target.elements[1].checked;
    const suggestionMsg = event.target.elements[3].value;
    const srcUrl = event.target.elements[4].value;

    fetch(feedbackUrl, {
      method: 'POST',
      headers: {
        'mode': 'cors',
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        created: new Date().toISOString(),
        page: srcUrl,
        comment: suggestionMsg,
        helpful: pageWasHelpful,
        user: 'anonymous'
      })
    }).then((response) => {
      if (response.status === 201) {
        displayFormMessage('Thanks for the Feedback!')
      } else {
        displayFormMessage(' An error occurred while submitting feedback.')
      }
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
