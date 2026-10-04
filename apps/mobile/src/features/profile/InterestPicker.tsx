import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../../hooks/useProfile";
import { MAX_INTERESTS } from "../../lib/validation";
import { spacing } from "../../theme";
import { Chip, ErrorState, Skeleton, Text } from "../../ui";

type Props = { selected: string[]; onChange: (slugs: string[]) => void };

export function InterestPicker({ selected, onChange }: Props) {
  const interests = useInterests();

  if (interests.isPending) {
    return (
      <View style={styles.wrap}>
        <Skeleton height={36} />
        <Skeleton height={36} width="70%" />
      </View>
    );
  }
  if (interests.isError) {
    return <ErrorState error={interests.error} onRetry={() => interests.refetch()} title="Couldn't load interests" />;
  }

  const full = selected.length >= MAX_INTERESTS;
  const toggle = (slug: string) =>
    onChange(selected.includes(slug) ? selected.filter((s) => s !== slug) : [...selected, slug]);

  return (
    <View style={styles.wrap}>
      <View style={styles.chips}>
        {interests.data.map((interest) => {
          const isSelected = selected.includes(interest.slug);
          return (
            <Chip
              key={interest.slug}
              disabled={!isSelected && full}
              label={interest.label}
              onPress={() => toggle(interest.slug)}
              selected={isSelected}
            />
          );
        })}
      </View>
      <Text tone="muted" variant="small">
        {selected.length} of {MAX_INTERESTS} selected
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.xs }
});
