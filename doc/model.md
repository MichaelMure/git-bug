# Data model

If you are not familiar with [git internals](https://git-scm.com/book/en/v1/Git-Internals), you might first want to read about them, as the `git-bug` data model is built on top of them.

The biggest problem when creating a distributed bug tracker is that there is no central authoritative server (doh!). This implies some constraints.

## Anybody can create and edit bugs at the same time as you

To deal with this problem, you need a way to merge these changes in a meaningful way.

Instead of storing the final bug data directly, we store a series of edit `Operation`s.

Note: In git-bug internally it is a golang struct, but in the git repo it is stored as JSON, as seen later.

These `Operation`s are aggregated in an `OperationPack`, a simple array. An `OperationPack` represents an edit session of a bug. We store this pack in git as a git `Blob`; that consists of a string containing a JSON array of operations. One such pack -- here with two operations -- might look like this:

```json
[
  {
    "type": "SET_TITLE",
    "author": {
      "id": "5034cd36acf1a2dadb52b2db17f620cc050eb65c"
    },
    "timestamp": 1533640589,
    "title": "This title is better"
  },
  {
    "type": "ADD_COMMENT",
    "author": {
      "id": "5034cd36acf1a2dadb52b2db17f620cc050eb65c"
    },
    "timestamp": 1533640612,
    "message": "A new comment"
  }
]
```

To reference our `OperationPack`, we create a git `Tree`; it references our `OperationPack` `Blob` under `"/ops"`. If any edit operation includes a media (for instance in a message), we can store that media as a `Blob` and reference it here under `"/media"`. 

To complete the picture, we create a git `Commit` that references our `Tree`. Each time we add more `Operation`s to our bug, we add a new `Commit` with the same data-structure to form a chain of `Commit`s.

This chain of `Commit`s is made available as a git `Reference` under `refs/bugs/<bug-id>`. We can later use this reference to push our data to a git remote. As git will push any data needed as well, everything will be pushed to the remote, including the media.

Here is the complete picture:

![git graph of a simple bug](bug-graph-1.png)

Now that we have this, we can easily merge our bugs without conflict. When pulling bug updates from a remote, we will simply add our new operations (that is, new `Commit`s), if any, at the end of the chain. In git terms, it's just a `rebase`.

// ----------------------TODO

## You can't have a simple consecutive index for your bugs

The same way git can't have a simple counter as identifier for its commits as SVN does, we can't have consecutive identifiers for bugs.

`git-bug` uses as identifier the hash of the first `Operation` of the bug, as serialized on disk. As this hash is ultimately computed with the content of the `CREATE` operation that includes title, message and a timestamp, it will be unique and prevent collision.

The same way as git does, this hash is displayed truncated to a 7 characters string to a human user. Note that when specifying a bug id in a command, you can enter as few characters as you want, as long as there is no ambiguity. If multiple bugs match your prefix, `git-bug` will complain and display the potential matches.

## You can't rely on the time provided by other people (their clock might be off) for anything other than just display

When in the context of a single bug, events are already ordered without the need of a timestamp. An `OperationPack` is an ordered array of operations. A chain of commits orders `OperationPack`s amongst each other.

Now, to be able to order bugs by creation or last edition time, `git-bug` uses a [Lamport logical clock](https://en.wikipedia.org/wiki/Lamport_timestamps). A Lamport clock is a simple counter of events. When a new bug is created, its creation time will be the highest time value we are aware of plus one. This declares a causality in the event and allows ordering bugs.

When bugs are pushed/pulled to a git remote, it might happen that bugs get the same logical time. This means that they were created or edited concurrently. In this case, `git-bug` will use the timestamp as a second layer of sorting. While the timestamp might be incorrect due to a badly set clock, the drift in sorting is bounded by the first sorting using the logical clock. That means that if users synchronize their bugs regularly, the timestamp will rarely be used, and should still provide a kinda accurate sorting when needed.

These clocks are stored in the chain of commits of each bug, as entries in each main git `Tree`. The first commit will have both a creation time and edit time clock, while a later commit will only have an edit time clock. A naive way could be to serialize the clock in a git `Blob` and reference it in the `Tree` as `"create-clock"` for example. The problem is that it would generate a lot of blobs that would need to be exchanged later for what is basically just a number.

Instead, the clock value is serialized directly in the `Tree` entry name (for example: `"create-clock-4"`). As a Tree entry needs to reference something, we reference the git `Blob` with an empty content. As all of these entries will reference the same `Blob`, no network transfer is needed as long as you already have any bug in your repository.


Example of Tree of the first commit of a bug:
```
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	create-clock-14
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	edit-clock-137
100644 blob a020a85baa788e12699a4d83dd735578f0d78c75	ops
```
Note that both `"ops"` and `"root"` entry reference the same OperationPack as it's the first commit in the chain.


Example of Tree of a later commit of a bug:
```
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	edit-clock-154
100644 blob 68383346c1a9503f28eec888efd300e9fc179ca0	ops
```
Note that the `"root"` entry still references the same root `OperationPack`. Also, all the clocks reference the same empty `Blob`.
