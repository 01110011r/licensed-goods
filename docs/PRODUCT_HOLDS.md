# How products are put on hold

This page explains when a product may be sold, which entities decide it, and how they relate. Use it as a reference when testing or explaining the system. The implementation is in `internal/licensing/matcher.go` (which groups a product needs) and `internal/catalogue/products.go` (whether the seller has the licences).

## Short version

A product can be sold when:

1. Its **status is `active`**, and
2. It is **not on hold**. It is on hold if it needs a licence group that its seller has no valid approved licence for.

Both checks are made **every time the product is read**. Nothing about the hold is stored on the product.

## The entities

| Entity | Key fields | What it means |
| --- | --- | --- |
| **Category** | `id`, `name`, `parent_id` | A node in a tree. Example: `c2` "Medicines" is the parent of `c3`. |
| **Licence group** | `id`, `name` | A kind of licence a product may need. Examples: `MED` "Medicines", `ALC` "Alcohol". |
| **Licence group rule** | `group_id`, `kind`, `category_id` or `prefix`, `excluded` | Says which products need a group. Kind is `category` or `ikpu`. |
| **Seller** | `id`, `name` | Owns products and holds licences. Example: `s1` "Shifo Apteka". |
| **Product** | `id`, `seller_id`, `category_id`, `ikpu`, `status` | The item being sold. Status is `draft`, `active` or `archived`. |
| **Licence** | `seller_id`, `group_id`, `status`, `valid_from`, `valid_to`, `file_path` | A seller's approval to sell a group's products. Status is `pending`, `approved` or `rejected`. |
| **Today** | `app_settings` row `today` | The application's date for all validity checks. It can differ from the real date. |

```text
                 ┌──────────── category rule ──────────┐
Category tree ◄──┤                                     ├── Licence group ◄── Licence ──► Seller
     ▲           └──── IKPU prefix rule (may exclude) ─┘                                    │
     │                                                                                     │ owns
     │ product's category                                                                  ▼
  Product ──────────────── needs ────────────────► required groups ◄── checked against ── seller's licences
  (seller, category, IKPU, status)                                       (approved and valid today)
```

## Step 1: which groups does a product need?

A product needs a group if **either** of these is true. The two checks are independent.

**Category rule.** The product's category, or any of its ancestors, has a category rule for the group. A rule on `c2` also covers `c3`, `c4` and so on below it. An unknown category matches no category rule.

**IKPU rule.** The product's 17-digit IKPU starts with an include prefix of the group. If several prefixes match, the longest one decides. If the longest is an exclusion, the group does not apply. On a tie in length, the exclusion wins.

A product can need several groups. It is on hold until the seller has a valid licence for **every** one.

## Step 2: does the seller have a valid licence for each group?

For each required group, the seller needs at least one licence where all of these are true:

- `status = approved`
- `valid_from ≤ Today`
- `valid_to` is empty, or `Today ≤ valid_to`

A licence that is `pending` or `rejected` does not count, and neither does an expired one. If any required group has no such licence, the product is **on hold**. The seller sees a reason such as `Missing valid approved licence for group ALC`.

## Step 3: who can see the product?

- **Buyer catalogue** (`GET /api/catalogue`): only products with `status = active` and **not** on hold.
- **Seller view** (`GET /api/sellers/{id}/products`): every product of that seller, with its hold status, reasons and any data issues.
- **Product list** (`GET /api/products`): every product, with the same details.

Held products are never removed from the seller's view. They are hidden only from buyers.

## When the answer changes

| Event | Effect |
| --- | --- |
| A product is added or its category or IKPU changes | Its required groups change on the next read. |
| A licence is submitted | Nothing changes. A pending licence does not count. |
| A licence is approved | Products needing that group come off hold on the next read, if the dates are valid today. |
| A licence is rejected | Nothing changes. |
| A licence's `valid_to` passes, or Today moves past it | Those products go back on hold on the next read. |
| Today is changed | Every licence's validity is recalculated on the next read. |
| A rule in a licence group is changed | Products matching the rule change hold status on the next read. |

## Licence lifecycle

1. The seller submits a licence for a group, with a document. It is `pending`.
   - A seller can have only one pending licence per group. A second submission is refused.
2. A reviewer approves it or rejects it with a reason.
   - Approval is refused if `valid_to` is before Today.
   - Rejection requires a reason.
3. An approved licence counts for every product that needs its group, while its dates are valid.

## Worked examples (seed data, Today = 2026-11-20)

| Product | Seller | Category | Needs | Seller's licence | Result |
| --- | --- | --- | --- | --- | --- |
| `p1` | s1 Shifo Apteka | c3 (under c2) | MED (category rule on c2) | MED approved, valid until 2027-01-10 | **Not held** |
| `p5` | s2 Vino Market | c8 | ALC (category rule on c8, and IKPU `2204`) | ALC approved, valid until 2026-11-25 | **Not held** |
| `p12` | s4 Universal Savdo | c8 | ALC | ALC **rejected** | **On hold** |
| (new) | s1 | c8 with IKPU `99999999999999999` | ALC (category rule only) | MED only, no ALC | **On hold** |
| (new) | s1 | a category and IKPU with no rules | nothing | none needed | **Not held** |

Products with status `draft` or `archived` are never shown to buyers, whatever their hold status.

## Things to know

- **Products that match no rule need no licence.** They are sellable when active. A mistyped category or IKPU can therefore silently remove a restriction.
- **Bad data does not block sale.** An invalid IKPU or unknown category adds a `data_issues` entry, but the product still counts as sellable if it is active and no group matches it.
- **No foreign keys.** The schema does not enforce that a product's seller or category exists. Application-level validation does this for products created through the API.
- **Today is not the real date.** Validity uses the `today` setting. Changing it lets you test expiry without waiting.

## How to test a hold by hand

1. Pick a seller and a group the seller has no approved licence for, for example seller `s1` and `ALC`.
2. Add a product whose category is covered by a rule of that group, for example category `c8`. Use an IKPU that matches no other rule. Set status `active`.
3. Check `GET /api/sellers/s1/products`. The product shows `on_hold: true` with the reason.
4. Check `GET /api/catalogue`. The product does not appear.
5. Optionally, submit and approve an ALC licence for `s1`. The product should come off hold.
