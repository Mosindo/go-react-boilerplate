import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterestsQuery } from "../hooks/useProfileData";
import { MAX_INTERESTS } from "../lib/validation";
import { Chip, Loader, Text, spacing } from "../shared/ui";

type InterestPickerProps = { selected: number[]; onChange: (ids: number[]) => void };

export function InterestPicker({ selected, onChange }: InterestPickerProps) {
  const interests = useInterestsQuery();
  if (interests.isLoading) {
    return <Loader size="small" />;
  }
  const toggle = (id: number) => {
    if (selected.includes(id)) {
      onChange(selected.filter((value) => value !== id));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, id]);
    }
  };
  return (
    <View style={styles.root}>
      <View style={styles.chips}>
        {(interests.data ?? []).map((interest) => (
          <Chip
            disabled={!selected.includes(interest.id) && selected.length >= MAX_INTERESTS}
            key={interest.id}
            label={interest.label}
            onPress={() => toggle(interest.id)}
            selected={selected.includes(interest.id)}
            testID={`interest-${interest.slug}`}
          />
        ))}
      </View>
      <Text tone="muted" variant="caption">
        {selected.length} / {MAX_INTERESTS} choisis
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
