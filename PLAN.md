# PLAN.md: Add Custom Frontmatter Fields to gonotes

## Overview

Add support for arbitrary YAML frontmatter fields via CLI flags, allowing users to add custom metadata beyond the standard `title`, `tags`, and `date` fields. This also renames the existing `-tm/-tr` flags to `-Tm/-Tr` for consistency.

## Background

gonotes currently supports these frontmatter fields:
- `title` - used for filename slug and symlinks
- `tags` - comma-separated, used for tag symlinks  
- `date` - used for date symlinks
- `ignore-links` - glob patterns for broken-link checking

The existing `-tm` (tag match) and `-tr` (tag replace) flags allow regex-based tag rewrites during note creation/update.

## Goals

1. Add `-Fk`/`-Fv` flags to set custom frontmatter key-value pairs
2. Add `-Fm`/`-Fr` flags for frontmatter regex-based rewriting (update only)
3. Rename `-tm/-tr` to `-Tm/-Tr` for consistency with new naming
4. Remove old `-tm/-tr` flags

## Changes

### 1. `prepare.go`

Add new types:
```go
type FrontmatterField struct {
    Key   string
    Value string
}

type FrontmatterRewrite struct {
    Match   string
    Replace string
}
```

Extend `PrepareOptions`:
```go
type PrepareOptions struct {
    Title              *string
    Tags               *string
    Date               *string
    TagRewrites       []TagRewrite      // existing
    ExtraFrontmatter   []FrontmatterField  // NEW
    FrontmatterRewrites []FrontmatterRewrite // NEW
    Now                func() time.Time
}
```

Update `Prepare()` function to:
1. Apply explicit Title/Tags/Date (existing)
2. Apply `ExtraFrontmatter`: iterate and call `note.Frontmatter.Set(f.Key, f.Value)`
3. Apply `FrontmatterRewrites`: for each rewrite, iterate frontmatter keys matching the regex, rename them (create new key, copy value, unset old key)

### 2. `cmd/gonotes/main.go`

#### `runNew` function:
- Add `-Tm` flag: "tag regex match (repeatable; pair with -Tr)"
- Add `-Tr` flag: "tag regex replace (repeatable; pair with -Tm)"
- Add `-Fk` flag: "set custom frontmatter key (repeatable; pair with -Fv)"
- Add `-Fv` flag: "set custom frontmatter value (repeatable; pair with -Fk)"
- **Remove** old `-tm/-tr` flags
- Add validation:
  - `-T` and `-Tm/-Tr` are mutually exclusive
  - `-Fk` and `-Fv` counts must be equal
  - `-Fk/-Fv` is combinable with `-T`

#### `runUpdate` function:
- Same `-Tm/-Tr`, `-Fk/-Fv` additions as `runNew`
- Add `-Fm` flag: "frontmatter key regex match (repeatable; pair with -Fr)"
- Add `-Fr` flag: "frontmatter key regex replace (repeatable; pair with -Fm)"
- Add validation:
  - `-T` and `-Tm/-Tr` are mutually exclusive
  - `-Fk` and `-Fv` counts must be equal

### 3. Tests

Add to `prepare_test.go`:
- `TestPrepareExtraFrontmatter`: verify custom fields are set correctly
- `TestPrepareFrontmatterRewrites`: verify key renaming via regex works

Add to CLI tests if applicable.

## Usage Examples

```bash
# Create note with custom fields
gonotes new -t "My Note" -T "tag1, tag2" -Fk href -Fv 'http://example.com' -Fk author -Fv 'Alice'

# Update note with custom fields
gonotes update -i 20260328-1 -Fk status -Fv 'published'

# Rename frontmatter key via regex (update only)
gonotes update -i 20260328-1 -Fm '^author$' -Fr 'written-by'

# Tag rewrite (renamed from -tm/-tr)
gonotes new -t "My Note" -Tm '^programming/' -Tr 'code/'
```

## Notes

- On `new`: only `-Fk/-Fv` is available (can't rewrite fields that don't exist)
- On `update`: both `-Fk/-Fv` and `-Fm/-Fr` can be used together
- Frontmatter rewrites operate on **keys** (not values), matching and renaming them
