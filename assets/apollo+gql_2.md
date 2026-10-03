# Apollo Client + GraphQL — complete KT (toddle web-app)

Two things, layered:
- **GraphQL** = the query *language* + the API contract (schema). Provider-agnostic.
- **Apollo Client** = the JS library that *runs* GraphQL from React, caches results, and re-renders components.

Toddle: `@apollo/client 3.9`, `graphql 15`. Everything under `src/lib/apolloClient/`.

---

# PART A — GraphQL (the language)

## 1. Why GraphQL vs REST
REST: many endpoints, fixed shapes, over/under-fetching.
```
GET /course/123        → whole course object (too much)
GET /course/123/subjects → another round trip (too little in one call)
```
GraphQL: **one endpoint**, you ask for the exact fields, get exactly that shape back.
```graphql
query { node(id:"123", type: COURSE) { id name subjects { id name } } }
```
One request, exact data, no over/under-fetch.

## 2. The three operation types
```graphql
query    { ... }   # READ  — fetch data
mutation { ... }   # WRITE — create/update/delete
subscription { ... }  # LIVE — server pushes updates (toddle uses Pusher instead, rarely GraphQL subs)
```

## 3. Anatomy of a query
```graphql
query getSubjectsOfGrade($id: ID!) {     # operation type + name + VARIABLES
  node(id: $id, type: GRADE) {           # a FIELD with ARGUMENTS
    id                                    # scalar field
    ... on Grade {                        # INLINE FRAGMENT (type-specific fields)
      id
      name
      subjects { id name isArchived }     # nested object fields
    }
  }
}
```
Parts:
- **Operation name** (`getSubjectsOfGrade`) — for debugging/caching.
- **Variables** (`$id: ID!`) — typed inputs. `!` = required (non-null).
- **Fields** — what you want back. Only what you ask for is returned.
- **Arguments** (`id: $id, type: GRADE`) — parameters on a field.
- **Inline fragment** (`... on Grade`) — see §6.

## 4. Variables & types
```graphql
query ($id: ID!, $filter: FilterInput, $count: Int = 10) { ... }
```
- Scalars: `ID`, `String`, `Int`, `Float`, `Boolean`, + enums (`GRADE`, `STUDENT`).
- `!` = non-null (required). `[Type!]!` = non-null list of non-null items.
- `= 10` = default value.
- Input objects (`FilterInput`) = structured argument types (used a lot in mutations).

Pass them from JS:
```js
useQuery(getSubjectsOfGrade, { variables: { id: "g1" } });
```

## 5. Fragments — reusable field sets
A fragment = a named bundle of fields you reuse across queries. Toddle has 200+ in `src/modules/CommonFragments.js`.
```graphql
fragment userNameFields on User {          # fragment ON a type
  id firstName lastName type
  ... on Student { preferredName pronouns }
}

query { node(id:$id, type: STAFF) { ...userNameFields } }   # spread it with ...
```
In toddle (JS interpolation):
```js
import { gql } from "lib/apolloClient";
export const userNameFragment = gql`
  fragment userNameFields on User { id firstName lastName type }
`;
export const q = gql`
  query($id:ID!){ node(id:$id,type:STAFF){ ...userNameFields } }
  ${userNameFragment}          // ← interpolate the fragment
`;
```
Why: DRY + Apollo caches by fragment shape. Change fields in one place.

## 6. `node(id, type)` + inline fragments — toddle's dominant shape
Toddle's schema exposes a generic `node(id, type)` entry point that returns an **interface/union**. You pick type-specific fields with `... on TypeName`:
```graphql
node(id: $id, type: GRADE) {
  id                          # common field (on the interface)
  ... on Grade { subjects { id } }      # only if it's a Grade
  ... on Subject { standards { id } }   # only if it's a Subject
}
```
This is why you see `... on X` everywhere. The server returns `__typename` so Apollo knows which it is.

## 7. Directives — `@skip` / `@include`
Conditionally include fields/branches based on a variable. Toddle uses this for subject-based vs course-based:
```graphql
query($isSubjectBasedGrading: Boolean!, $subjectId: ID, $learningCourseId: ID) {
  subjectDetails: node(id: $subjectId, type: SUBJECT) @include(if: $isSubjectBasedGrading) { ... }
  courseDetails:  node(id: $learningCourseId, type: LEARNING_COURSE) @skip(if: $isSubjectBasedGrading) { ... }
}
```
- `@include(if: X)` — include field only if X true
- `@skip(if: X)` — skip field if X true
One query, two shapes, switched by a boolean. (See gb.md Part 5.)

## 8. Aliases
Rename a field in the response (needed when you query the same field twice):
```graphql
{
  subjectDetails: node(id:$s, type:SUBJECT) { ... }   # alias → response.data.subjectDetails
  courseDetails:  node(id:$c, type:LEARNING_COURSE){...}
}
```

## 9. `__typename`
Apollo auto-adds `__typename` to every selection. It's the type name (`"Grade"`, `"Student"`). Critical because **cache normalization keys on `__typename + id`** (Part B §4).

## 10. Mutations
```graphql
mutation createRatings($input: [IBPYPElementRatingInput!]!) {
  planner {                                    # namespace
    createIBPYPElementRatings(input: $input) { # the mutation field
      id                                       # what to return after write
    }
  }
}
```
Same shape as a query but `mutation`. Returns data (usually the created/updated object) so the cache can update. Input is typically a typed **input object**.

---

# PART B — Apollo Client (the runtime)

## 1. The pieces
```
Component → useQuery/useMutation → ApolloClient → Link chain → network → GraphQL API
                                        ↓
                                   InMemoryCache (normalized store)
```

## 2. Client setup — `src/lib/apolloClient/apolloClientWrapper.js`
```js
new ApolloClientWrapper({
  link: createLinkChain(),   // authLink → sanitize → lsn → error → network
  cache,                     // InMemoryCache (below)
  defaultOptions: {
    watchQuery: { fetchPolicy: "cache-and-network", notifyOnNetworkStatusChange: true },
    query:      { fetchPolicy: "network-only" },
  },
});
```
Mounted once at app root: `<ApolloProvider client={client}>` (`src/main.js`).

## 3. Links — middleware for the network
A **link** is one step in a chain each request passes through. Like Redux middleware, but for GraphQL. Toddle chain:
- **AuthLink** — attach auth token/headers
- **SanitizeLink** — clean the request
- **LsnLink** — request tracking
- **ErrorLink** — catch errors → Sentry
- **NetworkLink** — the actual HTTP (splits BatchHttpLink vs HttpLink)

```js
// NetworkLink: batch multiple queries fired close together into ONE request
const defaultNetworkLink = split(
  op => op.getContext().useHttpLink === true,
  httpLink,        // plain, one request each
  batchHttpLink,   // BATCHED — fewer network calls
);
```
**BatchHttpLink** is why you see fewer requests in the Network tab than queries in code.

## 4. InMemoryCache — normalization (the heart)
Apollo stores every object **flat, keyed by `__typename:id`**. Nested queries get flattened into one big normalized table.
```
query result:  { course: { id:"c1", subjects:[{id:"s1"}, {id:"s2"}] } }

cache stores:
  "Course:c1":  { id:"c1", subjects:[ref("Subject:s1"), ref("Subject:s2")] }
  "Subject:s1": { id:"s1", ... }
  "Subject:s2": { id:"s2", ... }
```
Consequence: fetch `Subject:s1` in a DIFFERENT query later → **served from cache, no network.** And updating `Subject:s1` once updates it EVERYWHERE it's referenced. This is why you don't need Redux for server data.

Setup — `src/lib/apolloClient/Cache/index.js`:
```js
new InMemoryCacheWrapper({
  possibleTypes,      // ← union/interface type map (for ... on X)
  dataIdFromObject,   // ← custom cache-key function (normally __typename:id)
  typePolicies,       // ← per-type field rules (below)
});
```
- **`dataIdFromObject`** — how the cache key is built. Toddle customizes it.
- **`possibleTypes.json`** — tells the cache which concrete types implement each interface/union (needed because of `node` + `... on X`).
- **Rule: always request `id` in every object**, or normalization breaks (Apollo can't key it → data goes stale/duplicated).

## 5. typePolicies — per-type field behavior
50+ files in `src/lib/apolloClient/typePolicies/`. Mostly **`merge` functions for pagination** — how new page combines with cached page:
```js
// typePolicies/Subject.js
const SubjectTypePolicy = {
  Subject: { fields: { benchmarkLevels: { merge: mergeArrays } } },
};
```
Without a merge fn, fetching page 2 **replaces** page 1 (data lost). With it, they combine. **Pagination data wrong/duplicated → look here first.**

Also used for: computed local fields (`read`), custom key args (`keyArgs`).

## 6. fetchPolicy — cache vs network
Controls where data comes from:
| Policy | Behavior |
|--------|----------|
| `cache-first` (default) | cache if present, else network. Fastest |
| `cache-and-network` | show cache instantly, THEN refetch to update (toddle default for watchQuery) |
| `network-only` | always network, skip cache read |
| `no-cache` | network, don't even store result |
| `cache-only` | cache only, never network (throws if missing) |
| `standby` | like cache-first but doesn't auto-update |

`cache-and-network` = why UI **flashes old→new data**. Not a bug — showing stale cache then the fresh value. Key for "stale data" debugging.

## 7. Running a query — hooks (`useQuery`)
```js
import { useQuery } from "@apollo/client";  // (or the toddle barrel)

function SubjectList({ gradeId }) {
  const { data, loading, error, refetch, fetchMore } = useQuery(getSubjectsOfGrade, {
    variables: { id: gradeId },
    fetchPolicy: "cache-and-network",
    skip: !gradeId,                 // don't run if no id
    notifyOnNetworkStatusChange: true,
  });

  if (loading && !data) return <Spinner/>;
  if (error) return <Error e={error}/>;
  return data.node.subjects.map(s => <div key={s.id}>{s.name}</div>);
}
```
Returns:
- `data` — the result (undefined until first load; may hold cache while refetching)
- `loading` — boolean
- `error` — error object
- `refetch()` — re-run manually
- `fetchMore()` — pagination (append next page)

## 8. Running a query — the HOC (`graphql()`) — toddle's most common
~750 files. Wraps a component, injects result as a prop:
```js
import { compose, graphql } from "lib/apolloClient";

const withData = graphql(getAnnouncementPublishersQuery, {
  name: "getAnnouncementPublishers",             // prop name
  skip: ({ id }) => !id,
  options: ({ orgId, yearId }) => ({             // build variables from props
    fetchPolicy: "cache-and-network",
    variables: { orgId, yearId },
  }),
  props: ({ getAnnouncementPublishers }) => ({   // reshape into clean props
    publishers: getAnnouncementPublishers?.publishers ?? [],
    isLoading: getAnnouncementPublishers?.loading,
  }),
});

export default compose(connect(mapState), withData)(Container);
```
Same data as `useQuery`, delivered as props instead of a return value. Used because gradebook is class-component + compose code. (HOC vs hook = same job, see gb.md Part 1.)

## 9. Mutations — `useMutation`
```js
const [createRatings, { loading }] = useMutation(createRatingsMutation);

await createRatings({
  variables: { input: ratings },
  optimisticResponse: { ... },     // instant fake result (below)
  update: (cache, { data }) => {   // manually patch cache after write
    cache.modify({ ... });
  },
  refetchQueries: ["getStudentsRatings"],   // re-run these after
});
```

## 10. Optimistic UI + manual cache write (toddle's grade save)
Update the UI **instantly** before the server responds, roll back on error. Real toddle pattern (`StandardsGradebookModule.js`):
```js
// 1. write to cache immediately (optimistic) — UI updates now
writeStudentsRatingsForStandardsGradebookToCache({ variables, data: merged });

// 2. fire the mutation
const res = await client.mutate({ mutation: createGradebookRatingsMutation, variables: { input: ratings } });

// 3. on error → roll back the cache + toast
catch (e) {
  writeStudentsRatingsForStandardsGradebookToCache({ variables, data: original });
  dispatch(setToastMsg({ msg: e.message }));
}
```
Then a **2.5s debounced refetch** replaces the optimistic client-id with the real server id. This optimistic-write + debounced-refetch dance is where most gradebook bugs hide.

## 11. Reading/writing cache directly
```js
// read
const data = client.readQuery({ query: Q, variables });
// write
client.writeQuery({ query: Q, variables, data: newData });
// surgical field update
cache.modify({ id: cache.identify(obj), fields: { count: c => c + 1 } });
```
Toddle wraps these in helpers (`getXFromCache` / `writeXToCache`) — always follow the existing helper, don't hand-roll.

## 12. Refetching after a mutation
```js
useMutation(M, {
  refetchQueries: [{ query: Q, variables }],  // re-run these queries
  awaitRefetchQueries: true,                  // wait for them before resolving
});
```
Toddle uses debounced refetch instead of immediate, to batch rapid edits. **Don't add a naive refetch — it fights the debounced one.**

## 13. Pagination
```js
const { data, fetchMore } = useQuery(Q, { variables: { after: null } });

fetchMore({
  variables: { after: data.edges[data.edges.length-1].cursor },
  // merge handled by typePolicies field merge fn
});
```
Toddle schema uses **relay-style** `edges { node {...} cursor }` + `pageInfo`. Merging = the typePolicy `merge` functions.

## 14. Persistence
Cache persisted to **IndexedDB** via `apollo3-cache-persist` + `localForage`. Survives page refresh.
**Gotcha: stale IndexedDB cache can serve old data even after a code change.** Clear it when debugging weird cache bugs (Application tab → IndexedDB, or the persistor purge).

## 15. Error handling
- **Network errors** — request failed (no response). Caught in ErrorLink → Sentry.
- **GraphQL errors** — server responded with `errors[]` (bad query, resolver threw). In `error.graphQLErrors`.
```js
const { error } = useQuery(Q);
error?.networkError;      // transport failure
error?.graphQLErrors;     // per-field server errors
```

---

# PART B2 — the `graphql()` HOC in depth (toddle's main pattern)

The `graphql()` HOC is how ~750 toddle files run queries/mutations. Same data as `useQuery`/`useMutation`, but delivered as **props** by wrapping the component. Used because gradebook is class-component + `compose` code, and hooks can't run in classes or slot into `compose` chains.

## What it IS
`graphql(document, config)` = a **Higher-Order Component factory**. It returns an HOC (a function that takes a component and returns a wrapped one):
```js
graphql(QUERY, {...})            // → returns an HOC:  Component => WrappedComponent
graphql(QUERY, {...})(MyComp)    // → the wrapped component
```
Two calls, same shape as `connect` (see gb.md). It reads/writes via Apollo and injects the result as a prop.

## The config object — every option

```js
graphql(getCourseUserMapQuery, {
  name: "getCourseUserMap",                       // 1. prop name for the result
  skip: ({ courseId }) => _.isEmpty(courseId),    // 2. don't run when true
  options: (ownProps) => ({                        // 3. per-render query options
    fetchPolicy: "cache-and-network",
    variables: { id: ownProps.courseId, filters: {...} },
  }),
  props: ({ getCourseUserMap, ownProps }) => ({    // 4. reshape result into clean props
    students: parse(getCourseUserMap),
    isLoading: getCourseUserMap.loading,
  }),
})(Component);
```

### 1. `name` — what the result prop is called
Without `name`, the result lands on `props.data`. With `name: "getCourseUserMap"`, it lands on `props.getCourseUserMap`. **Always set `name`** when stacking multiple `graphql` (else they all fight over `props.data`).

The injected object has: `{ loading, error, variables, networkStatus, refetch, fetchMore, ...yourQueryFields }`.

### 2. `skip` — conditionally don't run
```js
skip: ({ courseId }) => _.isEmpty(courseId),   // no courseId → query never fires
```
Function of ownProps → boolean. Critical for queries that need a param that may not be ready yet. (Prevents firing with `undefined` variables.)

### 3. `options` — build variables + fetch behavior from props
```js
options: ({ courseId, academicYearId }) => ({
  fetchPolicy: "cache-and-network",
  variables: { id: courseId, filters: { academicYearIds: [academicYearId] } },
  pollInterval: 0,
  notifyOnNetworkStatusChange: true,
}),
```
Runs on every render → re-reads props → new variables → query re-fetches when variables change. This is the **reactive** link between props and the query.

### 4. `props` — reshape raw result into clean component props
```js
props({ getCourseUserMap: { variables, networkStatus }, ownProps }) {
  const courseData = getCourseUserMapFromCache(variables);   // read cache with the vars
  return {
    students: getStudentsDataMemoized({ courseData }),        // computed, memoized
    isLoading: _.includes([1, 2], networkStatus),             // networkStatus 1/2 = loading/setVariables
  };
}
```
Why: keeps the component dumb. Instead of `props.getCourseUserMap.data.node.subjects...` scattered in the component, `props` maps it ONCE to `props.students`. Also where memoized parsing + cache reads happen. `ownProps` = the props passed in from the parent/connect.

> `networkStatus` numbers: 1=loading, 2=setVariables, 3=fetchMore, 4=refetch, 6=poll, 7=ready, 8=error.

## Query HOC — full real example (`GradebookEnhancers.js:11`)
```js
export const withCourseUserMap = graphql(getCourseUserMapQuery, {
  name: "getCourseUserMap",
  skip: ({ courseId }) => _.isEmpty(courseId),
  options: ({ courseId, archivalState, userTypes, academicYearId }) => ({
    fetchPolicy: "cache-and-network",
    variables: { id: courseId, filters: { archivalState, userTypes: [userTypes], academicYearIds: academicYearId ? [academicYearId] : undefined } },
  }),
  props({ getCourseUserMap: { variables, networkStatus }, ownProps }) {
    const courseData = getCourseUserMapFromCache(variables);
    return {
      students: getStudentsSortedBySubjectLevel({ students: getStudentsDataMemoized({ courseData }) }),
      isData: !_.isEmpty(courseData),
      isLoading: _.includes([1, 2], networkStatus),
    };
  },
});
```
Now any component wrapped with `withCourseUserMap` just reads `props.students` / `props.isLoading`.

## Mutation HOC — full real example (`CircularCreateOrUpdateContainer.js:262`)
For mutations, `name` is a **function you call** (not data). The injected prop is the mutate fn; `props` usually wraps it with variables:
```js
graphql(editCircularMutation, {
  name: "updateCircular",
  skip: ({ mode }) => mode != "edit",
  props: ({ updateCircular, ownProps: { displayObject, userId } }) => ({
    // expose a clean fn the component calls: props.updateCircular(isPublished)
    updateCircular: isPublished =>
      updateCircular({                              // ← the raw mutate fn
        variables: { ...displayObject, updatedBy: userId, isPublished },
      }),
  }),
})
```
Component calls `this.props.updateCircular(true)` → fires the mutation. The `props` layer hides the variable-building.

## Stacking multiple — with `compose` (`CircularCreateOrUpdateContainer.js:144`)
```js
export default compose(
  connect(mapStateToProps, mapActionCreators),      // Redux
  graphql(getOrganizationCoursesQuery, { name: "getOrganizationCourses", options, props }),
  graphql(getStaffCoursesQuery,        { name: "getStaffCourses", options, props }),
  graphql(getCircularsDetailQuery,     { name: "getCircular", options, props }),
  graphql(editCircularMutation,        { name: "updateCircular", props }),   // mutation
  graphql(createCircularMutation,      { name: "createCircular", props }),   // mutation
)(CircularCreateOrUpdate);
```
Each `graphql` adds its prop to the box; `compose` stacks them (see gb.md Part 2 for the compose/render walkthrough). Order matters if a later HOC's `options` needs a prop an earlier HOC produced.

## How it works at render (mechanics)
1. Wrapped component renders.
2. HOC runs `skip(ownProps)` — if true, injects a no-op prop, no network.
3. Else runs `options(ownProps)` → gets `{ variables, fetchPolicy }`.
4. Subscribes to that query in the Apollo cache (like a `watchQuery`).
5. On cache/network change → re-runs `props(...)` → injects fresh props → component re-renders.
6. On unmount → unsubscribes (no leak — same lifecycle as connect, gb.md Part 3).

So `graphql()` HOC = **Apollo's `useQuery` packaged as a wrapper**, exactly as `connect` is Redux's `useSelector` packaged as a wrapper.

## Rules for using the `graphql()` HOC (toddle)
1. **Always set `name`** — avoids `props.data` collisions when stacking.
2. **Always `skip`** when a required variable can be missing — never fire with `undefined` vars.
3. **Put variable-building in `options`**, not the component.
4. **Put parsing/reshaping in `props`**, not the component — keep components dumb; memoize heavy parses.
5. **Import `graphql` from the barrel** `lib/apolloClient`, not `@apollo/client`.
6. **Stack via `compose`**, mind the order (later HOCs can read earlier HOCs' props).
7. **Mutations:** `name` becomes a callable fn; wrap it in `props` to inject clean variables.
8. **New code → prefer hooks** (`useQuery`/`useMutation`). Only use the HOC when the file is a class component or already a `compose` chain (match the file — gb.md coding norm).
9. Convention: query HOCs live in `*Enhancers.js` files, exported as `withXxx`, then composed onto containers.

## HOC vs hook — same job (toddle has both)
```js
// HOC (old, gradebook)                          // HOOK (new)
graphql(Q, { name:"getX", options, props })(C)   const { data, loading } = useQuery(Q, { variables });
// → props.getX.data, props.getX.loading         // → data, loading (named yourself)
graphql(M, { name:"doX", props })(C)             const [doX] = useMutation(M);
// → this.props.doX(args)                         // → doX({ variables })
```
Same Apollo underneath. Wrap-and-inject vs call-and-return.

---

# PART B3 — Apollo hooks in depth (the modern way)

Hooks = the newer way to run GraphQL, used in ~665 toddle files. Same Apollo underneath as the `graphql()` HOC — but called INSIDE a function component, returning values instead of injecting props. Can't be used in class components or `compose` chains (that's why gradebook still uses the HOC).

## `useQuery` — read data

```js
import { useQuery } from "@apollo/client";   // (or the toddle barrel)

const { data, loading, error, networkStatus, refetch, fetchMore, previousData } =
  useQuery(getStudentSessionsQuery, {
    variables: sessionsVariables,        // the query variables
    skip: !studentId,                    // don't run when true
    fetchPolicy: "cache-and-network",    // cache vs network
    context: { useHttpLink: true },      // per-request link context (toddle: force non-batched)
    notifyOnNetworkStatusChange: true,   // re-render on refetch/poll (so `loading` flips)
    pollInterval: 0,                     // >0 = auto-refetch every N ms
    onCompleted: (data) => {},           // callback on success
    onError: (err) => {},                // callback on error
  });
```

### Return values
| Field | Is |
|-------|-----|
| `data` | result (undefined until first load; may hold cache while refetching) |
| `previousData` | last successful data (useful during refetch to avoid flicker) |
| `loading` | boolean |
| `error` | `{ networkError, graphQLErrors }` |
| `networkStatus` | 1=loading 2=setVariables 3=fetchMore 4=refetch 6=poll 7=ready 8=error |
| `refetch(vars?)` | re-run manually |
| `fetchMore({variables})` | pagination — append next page |

### Real toddle example (`useSessions.js:192`)
```js
const { data: sessionsData, networkStatus } = useQuery(getStudentSessionsQuery, {
  fetchPolicy: "cache-and-network",
  skip: !studentId || !selectedAcademicYearId,
  variables: sessionsVariables,
  context: { useHttpLink: true },
});

// parse with useMemo — the hooks equivalent of the HOC's `props` reshaping
const rawEdges = useMemo(
  () => _.get(sessionsData, "node.attendanceActivities.edges", EMPTY_ARRAY),
  [sessionsData]
);
```
**Note the pattern:** `useQuery` gives raw `data`; you reshape with `useMemo` (memoized, like the HOC's `props`). Reshaping doesn't disappear — it just moves from `props:` into `useMemo`.

## `useMutation` — write data
```js
const [createLevel, { loading, error, data }] = useMutation(createAcademicSubjectLevelMutation, {
  refetchQueries: [{ query: getLevelsQuery, variables }],  // re-run after
  awaitRefetchQueries: true,
  optimisticResponse: {...},          // instant fake result (see Part B §10)
  update: (cache, { data }) => {...}, // manually patch cache after write
  onCompleted: () => {},
  onError: () => {},
});

// call it later, pass variables at call time:
await createLevel({ variables: { input } });
```
Returns a **tuple**: `[mutateFn, resultState]`. Unlike `useQuery`, it does NOT run on render — only when you call `mutateFn`. Real: `LevelsV2.js:249` (`const [createAcademicSubjectLevel] = useMutation(...)`).

## `useLazyQuery` — query on demand (not on render)
When you want a query to run on an event (click, submit), not automatically:
```js
const [loadStandards, { data, loading }] = useLazyQuery(getStandardsQuery);
// later:
onClick={() => loadStandards({ variables: { subjectId } })}
```
Tuple like `useMutation`, but for reads. Use when the trigger is user action, not mount.

## `useApolloClient` — the raw client
Get the client instance to do imperative reads/writes outside a query:
```js
const client = useApolloClient();
const cached = client.readQuery({ query: Q, variables });
client.writeQuery({ query: Q, variables, data });
await client.mutate({ mutation: M, variables });
```
Toddle also exposes the client to non-React code via `apolloClientAccessor` (`lib/apolloClient/client`) — for cache helpers that can't use hooks.

## `useFragment` (AC3.8+) — subscribe to one cached object
Read a single normalized object and re-render only when IT changes (not the whole query):
```js
const { data } = useFragment({ fragment: userNameFragment, from: { __typename: "User", id } });
```
Rare in toddle today but the modern granular-read tool.

## `useSubscription` — live server pushes
```js
const { data } = useSubscription(onGradeChangedSub, { variables });
```
Rare — toddle uses **Pusher** for realtime, not GraphQL subscriptions.

## The toddle custom hook wrappers
Toddle wraps Apollo reads in its own hooks/HOCs (`lib/apolloClient/hoc/`): `withReadQuery`, `withBackgroundQuery`, plus `backgroundQueryProvider`. And feature-level custom hooks (`useSessions.js`, `useStudentAttendanceOverview.js`) bundle a `useQuery` + parsing + filters into one reusable hook — the modern replacement for `*Enhancers.js` HOCs.

## Rules for hooks (toddle)
1. **Hooks rules apply** — call at top level of the component/custom hook, never in loops/conditions/after early return.
2. **`skip`** instead of conditionally calling — never `if (x) useQuery(...)`. Use `useQuery(Q, { skip: !x })`.
3. **Reshape with `useMemo`** keyed on `data` — don't recompute every render, don't build new arrays inline (causes re-render storms downstream).
4. **`variables` object stability** — inline object literals are new every render; useMemo the variables if they're heavy, or Apollo may over-refetch.
5. **Mutations don't run on render** — you must call the returned fn.
6. **New code → hooks.** Editing HOC/class code → stay HOC (match the file, gb.md coding norm).
7. Wrap reusable query+parse logic in a **custom hook** (`useXxx`), the hooks-era analog of `withXxx` enhancers.

## HOC vs hook — the same three jobs
| Job | HOC (`graphql`) | Hook (`useQuery`) |
|-----|-----------------|-------------------|
| variables | `options: p => ({variables})` | `{ variables }` arg |
| skip | `skip: p => bool` | `{ skip: bool }` |
| reshape result | `props: r => ({...})` | `useMemo(() => ..., [data])` |
| result delivery | injected as props | returned values |
| where usable | any component (incl. class) + compose | function components only |

Same Apollo engine. Wrap-and-inject vs call-and-return.

---

# PART C — Toddle-specific rules & gotchas

1. **Import from the barrel** `lib/apolloClient`, NOT `@apollo/client` directly — it wraps/patches. `import { gql, graphql, compose } from "lib/apolloClient"`.
2. **Queries colocated** in `*Query.js` / `*Queries.js` / `*Mutations.js` / `*Fragments.js` (~1400 files). `babel-plugin-graphql-tag` compiles `gql` at build time.
3. **`node(id, type)` + `... on Type`** everywhere. Learn it.
4. **Always request `id`** in every object (normalization).
5. **Fragments centralized** in `src/modules/CommonFragments.js` (200+).
6. **`graphql-request`** (not Apollo) used in a few spots for one-off calls outside React.
7. **`apolloClientAccessor.js`** — a Proxy exposing the client to non-React modules (cache helpers) without circular imports. Import as `client` from `lib/apolloClient/client`.
8. **cache-and-network default** → expect the old→new flash.
9. **Optimistic + debounced refetch** on writes → understand `mergeXForCacheUpdate` before touching save logic.

---

# PART D — Debugging Apollo

**Locate the layer first:**
1. Wrong data on screen → **Apollo DevTools** → inspect cache + run the query live.
   - Right in cache, wrong on screen → component/props/selector bug.
   - Wrong in cache → query or backend.
2. **Network tab** → find the GraphQL request → check response JSON.
   - Wrong there → backend/query.
   - Right there but wrong in cache → typePolicy `merge` / normalization / missing `id`.
3. Stale after refresh → persisted IndexedDB cache. Clear it.
4. Old→new flash → `cache-and-network`, working as designed.
5. Pagination duplicated/lost → typePolicy `merge` function.
6. Data from another query unexpectedly changed → normalization (shared `__typename:id` object updated everywhere).

Tools: **Apollo DevTools** (cache explorer, run queries, watch active queries), Network tab, `client.cache.extract()` in console (dump whole cache).

---

# Quick reference — GraphQL + Apollo

| Concept | What | Toddle |
|---------|------|--------|
| query / mutation | read / write | `*Query.js` / `*Mutations.js` |
| variables `$x: ID!` | typed inputs | `options: () => ({ variables })` |
| fragment | reusable fields | `CommonFragments.js` (200+) |
| `node(id,type)` + `... on X` | generic entry + type fields | dominant shape |
| `@skip`/`@include` | conditional fields | subject vs course query |
| InMemoryCache | normalized store by `__typename:id` | `Cache/index.js` |
| typePolicies | per-type field/merge rules | `typePolicies/` (50+) |
| fetchPolicy | cache vs network | `cache-and-network` default |
| useQuery | run query (hook) | ~665 files |
| graphql() HOC | run query (wrapper) | ~750 files (gradebook) |
| useMutation | write | grade save |
| optimisticResponse | instant UI | grade save + debounced refetch |
| links | network middleware | auth/error/batch |
| persistence | IndexedDB | apollo3-cache-persist |

## Key files
| Purpose | Path |
|---------|------|
| Client | `src/lib/apolloClient/apolloClientWrapper.js` |
| Cache + typePolicies | `src/lib/apolloClient/Cache/index.js`, `src/lib/apolloClient/typePolicies/` |
| Links | `src/lib/apolloClient/Middlewares/` |
| gql barrel (import here) | `src/lib/apolloClient/index.js` |
| Client accessor (non-React) | `src/lib/apolloClient/client` (apolloClientAccessor.js) |
| Common fragments | `src/modules/CommonFragments.js` |
| Example query | `src/CommonGraphql/Grade/GradeQueries.js` |
| Example HOC usage | `src/AppComponents/Announcements/modules/AnnouncementEnhancers.js` |
| Optimistic mutation | `src/AppComponents/Gradebook/routes/Standards/modules/StandardsGradebookModule.js` |
