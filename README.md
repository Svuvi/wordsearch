My first Golang (no frameworks) + HTMX webapp. Glorified todo list app, with storing and active word searching from a database.
I actually ended up using it myself to aid me in learning the Dutch language.

What can it do:
* CRUD words
* Active search and search higlighting
* Session token authroisation written from scratch

I have a list of things that could be improved and features that might be added to this app.
I might or might not introduce them in the future. Ordered from more relevant to less relevant:

* First and the most huge: Client-side table search and filtering
* Password change
* Click to edit for table cells
* Caching for isAuthorised function
* A way to restore password using email
* Export and import of a table in JSON format
* JSON api for a hypothetical mobile app or integration
* Making a decent style for the login page

There are also issues that im aware of:
* New registrations aren't capped in any way, there is no anti-bot features/defence
* No database entries cap for users
* No hard limits on the length of the strings being put in the database
* XSS is possible if the user's account is compromised.
  This is because I used text/template instead of html/template, so it doesnt escape search highlighting. This also means it won't escape an injected script element