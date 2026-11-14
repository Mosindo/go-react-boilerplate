import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme/palette";
import { useInterests } from "../../state/queries";
import { Chip, ErrorState, LoadingState, Text } from "../../ui";

export const MAX_INTERESTS = 10;

export function InterestPicker({ value, onChange }: { value: string[]; onChange: (next: string[]) => void }) {
  const { data, isLoading, isError, refetch } = useInterests();
  const groups = useMemo(() => {
    const map = new Map<string, NonNullable<typeof data>>();
    for (const i of data ?? []) {
      map.set(i.category, [...(map.get(i.category) ?? []), i]);
    }
    return [...map.entries()];
  }, [data]);

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Impossible de charger les centres d'intérêt." onRetry={refetch} />;

  const toggle = (slug: string) => onChange(value.includes(slug) ? value.filter((s) => s !== slug) : [...value, slug]);
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="caption">
        {value.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      {groups.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <Text tone="subtle" variant="overline">
            {category}
          </Text>
          <View style={styles.chips}>
            {items.map((i) => (
              <Chip
                disabled={!value.includes(i.slug) && value.length >= MAX_INTERESTS}
                key={i.slug}
                label={i.label}
                onPress={() => toggle(i.slug)}
                selected={value.includes(i.slug)}
              />
            ))}
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
