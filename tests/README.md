README

Playground tests 


Suite#1: Browser automated selenium tests, which runs all examples for all languages.

    Runtime Steps: (~10-15mins)
   1) pip3 install -r requirements.txt
   
   2) Download Chrome browser webdriver ChromeDriver 92.0.4515.107 from https://sites.google.com/chromium.org/driver/downloads
       
       $ mv chromedriver /usr/local/bin/ or cp chromedriver /usr/local/bin/
   
   3) Run the tests

     To run on couchbase.live:
     $ python3 cblive_playground_browsertest.py

     To run on a specific URL:
     $ CBLIVE_URL="http://cb-84172.couchbase.live" python3 cblive_playground_browsertest.py
    
     See the chrome browser automatically coming and running the clicks. At the end it will show the summary counts. F --> Fail and E --> Error. Errors are something on the test code need to be fixed. Fails are the issues. On the console, you can see the output extracted from code runtime.

    
    Running in Headless mode (non GUI):
     DRIVER_OPTIONS='-headless' python cblive_playground_browsertest.py

    Running with other driver options settings including executable driver path:
       DRIVER_OPTIONS='executable_path=/usr/local/bin/chromedriver,-headless' python cblive_playground_browsertest.py


Suite#2: Running all the examples code from the .yaml files using /run requests.
    Runtime steps:
    
    $ pip3 install -r requirements.txt
    
    $ CBLIVE_URL=https://couchbase.live CODE_DIR=cmd/play-server/static/examples CB_USER=Administrator CB_PWD=${CB_ADMIN_PASSWORD} \
		python tests/cblive_playground_runcodetest.py
		
    $ CBLIVE_URL=http://localhost:8080 CODE_DIR=cmd/play-server/static/examples \
		python tests/cblive_playground_runcodetest.py
