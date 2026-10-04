import React from "react";
import { FlatList, Pressable, StyleSheet } from "react-native";
import type { MatchSummary } from "../../api/types";
import { spacing } from "../../theme";
import { ProfilePhoto, Text } from "../../ui";

type Props = { matches: MatchSummary[]; onPress: (match: MatchSummary) => void };

/** Horizontal strip of matches nobody has written to yet. */
export function NewMatchesRow({ matches, onPress }: Props) {
  if (matches.length === 0) return null;
  return (
    <>
      <Text accessibilityRole="header" style={styles.title} tone="muted" variant="small" weight="semibold">
        NEW MATCHES
      </Text>
      <FlatList
        contentContainerStyle={styles.list}
        data={matches}
        horizontal
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <Pressable
            accessibilityHint="Opens the chat"
            accessibilityLabel={`New match: ${item.user.displayName}`}
            accessibilityRole="button"
            onPress={() => onPress(item)}
            style={styles.item}
          >
            <ProfilePhoto
              name={item.user.displayName}
              photo={item.user.photo}
              round
              size="thumb"
              style={styles.avatar}
            />
            <Text numberOfLines={1} style={styles.name} variant="small" weight="medium">
              {item.user.displayName}
            </Text>
          </Pressable>
        )}
        showsHorizontalScrollIndicator={false}
      />
    </>
  );
}

const styles = StyleSheet.create({
  title: { paddingHorizontal: spacing.md, paddingTop: spacing.sm, letterSpacing: 0.6 },
  list: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, gap: spacing.md },
  item: { width: 72, alignItems: "center", gap: 4 },
  avatar: { width: 68, height: 68, borderWidth: 2, borderColor: "#B8472F", borderRadius: 34 },
  name: { maxWidth: 72 }
});
