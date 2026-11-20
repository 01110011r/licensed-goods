# Test task: licensed goods

## The story

You are building the piece of an online marketplace that decides which products may be sold. Time budget: 2 working days.

Some goods can only be sold by a seller who holds a licence: medicines, alcohol, pesticides and so on. A seller lists a product in a **category**, and every product also has a 17-digit government product code (**IKPU**). Each **licence group** has rules that say which products need it:

- **Category rule**: the group covers a category and every category below it in the tree.
- **Code rule**: the group covers every product whose IKPU starts with a given prefix. A rule can also **exclude** a prefix. When several code rules match, the one with the longest prefix wins.

A seller uploads a licence for a group. A reviewer on the marketplace staff approves or rejects it. A product that needs a group the seller has no valid licence for is **on hold**: buyers can't see it, but the seller still sees it with the reason. Being on hold is a separate flag, not a product status: the product keeps its own status (draft, active, archived).

You build the tool for sellers and reviewers: a page where a seller sees why each product is on hold and sends a licence, a page where a reviewer goes through the queue, and the backend that checks every rule.

## What to build

A backend and a web page. Both are required.

**The page**

1. **Seller: my products.** Pick a seller from a select (no login). Show every product with its category, IKPU, status and whether it is on hold. For a held product say why in plain words, for example "Needs a Medicines licence. Yours expired on 1 November."
2. **Seller: send a licence.** A form with group, licence number, issuer, valid from, valid to (optional: some licences never expire) and a file. Check the fields in the page and again on the server.
3. **Reviewer: queue.** Licences waiting for review, oldest first. Approve, or reject with a reason the seller will see. Show which products would come off hold if this licence is approved.
4. **Buyer: catalogue.** The products buyers can see right now. Held products must not appear here.
5. **Today.** A date picker that sets "today" for the whole app, so the reviewer can check what happens when licences expire.

**The server** stores everything and enforces every rule. Checks in the page are only for comfort: a request sent by hand must be refused too.

Use any language. Go for the backend and React for the page is a plus, since that is what we use, but it is not required. Keeping data in memory or in SQLite is fine, and losing changes on restart is fine. Store the file anywhere (disk is fine); we only check that it can be opened again.

## Rules

- A product needs a group when a category rule or a code rule of that group matches it. A product can need more than one group, and it is on hold until the seller has a valid licence for **every** one.
- A licence is valid when it is approved and today is between valid from and valid to. A licence without valid to never expires.
- Approving a licence takes the matching products off hold at once. A licence that expires, or is revoked, puts them back on hold at once. No product status changes in either case.
- A seller has at most one licence waiting for review per group. A new one for the same group replaces the old one only after it is approved; until then the old one keeps working.
- A rejected licence must have a reason. A reviewer can't approve a licence with valid to in the past.
- Some products and licences in the seed data are broken on purpose. Find them, show them with the reason, and make sure nothing crashes on them.

If something is unclear, decide, write the decision in your README and keep going.

## Seed data

`data.json` comes with this task. Load it on start: categories (a tree), licence groups with their rules, sellers, products and some licences already sent. Use `today` from the file as the starting date.

You design the API between the page and the server yourself.

## Bonus

Only once the required parts work. One done well beats three half done.

- **Expiry reminders.** A list of licences that expire in the next 14 days, with the products that will go on hold.
- **History.** Every change to a licence (sent, approved, rejected, expired, revoked) with who and when, shown on the licence.
- **Two-person rules.** A reviewer can draft a new group rule, but a second reviewer must publish it. Before publishing, show how many products it will put on hold.
- **Docker.** One command starts everything.

## Hand-in and scoring

Send a link to a git repo. It must contain:

- **README**: how to run it in under five commands, every assumption you made, what you skipped and why, how long it took, and how you used AI tools, if at all
- **Tests** for the server rules: category and code rules (including longest prefix and exclude), every way a licence can be refused, approve releases products, expiry holds them again
- **Commit history** as you worked. One commit at the end loses points

We then run it, try to break it from the page and with curl, and talk with you for 30 minutes about one decision you made.

| What we check | Points |
| --- | --- |
| Hold rules: the server decides correctly which products need which licence, and the page explains it in plain words | 25 |
| Send licence: checked in the page and on the server, file stored | 15 |
| Reviewer queue: approve and reject work, products come off hold at once | 20 |
| Changing "today" expires licences and holds products again; the catalogue never shows a held product | 10 |
| Broken data shown with its reason; nothing crashes on it | 10 |
| Tests, README and commits | 10 |
| Code is easy to read and change | 10 |
| Bonus: expiry reminders +5, history +5, two-person rules +10, anything else +5 | up to +20 |
