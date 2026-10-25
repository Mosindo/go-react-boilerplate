import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import type { MatchItem } from "../../api";
import { Avatar, Text } from "../../shared/ui";
import { MIN_TOUCH_TARGET, spacing, useTheme } from "../../theme";

type Props = { matches: MatchItem[]; onOpen: (match: MatchItem) => void; onEndReached: () => void };

const NewMatch = React.memo(function NewMatch({
  match,
  onOpen
}: {
  match: MatchItem;
  onOpen: (match: MatchItem) => void;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityHint="Ouvre la conversation"
      accessibilityLabel={`Nouveau match avec ${match.profile.firstName}`}
      accessibilityRole="button"
      onPress={() => onOpen(match)}
      style={styles.item}
      testID={`new-match-${match.matchId}`}
    >
      <View style={[styles.ring, { borderColor: colors.primary }]}>
        <Avatar size={64} thumbUrl={match.profile.photos[0]?.thumbUrl} />
      </View>
      <Text numberOfLines={1} style={styles.name} variant="caption">
        {match.profile.firstName}
      </Text>
    </Pressable>
  );
});

export function NewMatchesStrip({ matches, onOpen, onEndReached }: Props) {
  return (
    <View style={styles.section}>
      <Text variant="heading">Nouveaux matchs</Text>
      <FlatList
        contentContainerStyle={styles.row}
        data={matches}
        horizontal
        keyExtractor={(match) => match.matchId}
        onEndReached={onEndReached}
        renderItem={({ item }) => <NewMatch match={item} onOpen={onOpen} />}
        showsHorizontalScrollIndicator={false}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: spacing.md },
  row: { gap: spacing.lg },
  item: { alignItems: "center", gap: spacing.xs, minWidth: MIN_TOUCH_TARGET + 24 },
  ring: { borderWidth: 2, borderRadius: 40, padding: 3 },
  name: { maxWidth: 76 }
});
