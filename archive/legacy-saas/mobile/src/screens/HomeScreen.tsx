import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { createPost, listPosts, type Post } from "../api/platform";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Badge, Button, Card, Input, Notice, Text, colors, spacing } from "../shared/ui";
import { Header, ScreenContainer } from "../shared/layout";

type HomeScreenProps = {
  token: string;
  currentUserId: string;
};

function formatPostDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown date";
  }
  return date.toLocaleString();
}

export default function HomeScreen({ token, currentUserId }: HomeScreenProps) {
  const [posts, setPosts] = useState<Post[]>([]);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const nextPosts = await listPosts(token);
      setPosts(nextPosts);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "could not load posts");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  async function onSubmit() {
    const normalizedTitle = title.trim();
    const normalizedBody = body.trim();
    if (!normalizedTitle || !normalizedBody) {
      setError("title and body are required");
      return;
    }

    setSubmitting(true);
    setError(null);
    setFeedback(null);
    try {
      const created = await createPost(token, normalizedTitle, normalizedBody);
      setPosts((prev) => [created, ...prev.filter((item) => item.id !== created.id)]);
      setTitle("");
      setBody("");
      setFeedback("Post published.");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "could not create post");
    } finally {
      setSubmitting(false);
    }
  }

  const emptyState = useMemo(() => {
    if (loading) {
      return null;
    }
    return (
      <EmptyView
        message="Share progress, notes, or the first workspace announcement."
        title="No posts yet"
      />
    );
  }, [loading]);

  if (loading && posts.length === 0) {
    return (
      <ScreenContainer testID="home-screen">
        <LoadingView fullScreen label="Loading workspace..." />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer testID="home-screen">
      <Header
        action={
          <Button
            disabled={loading || submitting}
            label="Reload"
            onPress={load}
            size="sm"
            testID="home-reload-button"
            variant="outline"
          />
        }
        eyebrow="Workspace"
        style={styles.header}
        subtitle="Team updates, product notes, and lightweight announcements."
        title="Home"
      />

      <Card style={styles.composer} variant="accent">
        <Text style={styles.sectionTitle} variant="heading" weight="bold">
          Publish an update
        </Text>
        <Input
          containerStyle={styles.field}
          label="Title"
          onChangeText={setTitle}
          placeholder="Post title"
          testID="home-post-title-input"
          value={title}
        />
        <Input
          containerStyle={styles.field}
          label="Body"
          multiline
          numberOfLines={4}
          onChangeText={setBody}
          placeholder="Share progress, notes, or an announcement..."
          style={styles.bodyInput}
          testID="home-post-body-input"
          value={body}
        />
        <Button
          fullWidth
          disabled={loading}
          label="Publish"
          loading={submitting}
          onPress={onSubmit}
          testID="home-post-submit-button"
        />
      </Card>

      {error ? (
        <ErrorView actionLabel="Retry" message={error} onAction={() => void load()} style={styles.error} />
      ) : null}
      {feedback ? (
        <Notice description="Your update is now visible in the workspace feed." style={styles.feedback} title={feedback} tone="success" />
      ) : null}

      <FlatList
        contentContainerStyle={styles.list}
        data={posts}
        keyExtractor={(item) => item.id}
        ListEmptyComponent={emptyState}
        renderItem={({ item }) => (
          <Card style={styles.card}>
            <View style={styles.cardHeader}>
              <Text style={styles.cardTitle} variant="heading" weight="bold">
                {item.title}
              </Text>
              <Badge label={item.authorUserId === currentUserId ? "You" : "Member"} size="sm" variant="primary" />
            </View>
            <Text style={styles.cardBody}>{item.body}</Text>
            <Text style={styles.meta} tone="muted" variant="caption">
              {formatPostDate(item.createdAt)}
            </Text>
          </Card>
        )}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  header: {
    marginBottom: spacing.lg
  },
  composer: {
    marginBottom: spacing.md
  },
  sectionTitle: {
    marginBottom: spacing.lg
  },
  field: {
    marginBottom: spacing.md
  },
  bodyInput: {
    minHeight: 120
  },
  error: {
    marginBottom: spacing.sm
  },
  feedback: {
    marginBottom: spacing.sm
  },
  list: {
    paddingBottom: spacing.xxxl
  },
  empty: {
    marginTop: spacing.xxxl
  },
  card: {
    marginBottom: spacing.md
  },
  cardHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: spacing.sm,
    gap: spacing.sm
  },
  cardTitle: {
    flex: 1,
    color: colors.text
  },
  cardBody: {
    color: colors.text,
    lineHeight: 22
  },
  meta: {
    marginTop: spacing.md
  }
});
