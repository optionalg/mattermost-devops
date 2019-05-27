# On Off Boarding

This Lambda function listen the Onelogin Webhook Events and when some specific events show up it react.

This Lambda is to take care the on and off boarding the Mattermost employees. It wait for the creation onelogin account
to add the user in the github if need and the deactivation/unlicensing/suspend user to remove from github team.


TODO:
  - After removed we can add a call to delete the user from Onelogin.

