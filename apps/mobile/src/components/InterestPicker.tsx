import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../hooks/queries";
import { Chip, ErrorView, LoadingView, Text, spacing } from "../ui";

const MAX_INTERESTS = 10;

type Props = { selected: string[]; onChange: (slugs: string[]) => void };

export function InterestPicker({ selected, onChange }: Props) {
  const query = useInterests();

  if (query.isLoading) return <LoadingView />;
  if (query.isError)
    return <ErrorView message="Impossible de charger les centres d'intérêt." onRetry={() => void query.refetch()} />;

  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, slug]);
    }
  };

  return (
    <View style={styles.wrap}>
      <View style={styles.chips}>
        {(query.data ?? []).map((interest) => (
          <Chip
            key={interest.slug}
            label={interest.label}
            onPress={() => toggle(interest.slug)}
            selected={selected.includes(interest.slug)}
          />
        ))}
      </View>
      <Text variant="caption" tone="muted">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
