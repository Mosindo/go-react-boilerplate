import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../../../hooks/queries";
import { spacing } from "../../../theme/palette";
import { Chip, Skeleton, Text, TextField } from "../../../ui";

type Props = {
  bio: string;
  interestIds: number[];
  onBioChange: (bio: string) => void;
  onInterestsChange: (ids: number[]) => void;
};

export const MAX_INTERESTS = 10;

export function AboutForm({ bio, interestIds, onBioChange, onInterestsChange }: Props) {
  const interests = useInterests();
  const toggle = (id: number) => {
    if (interestIds.includes(id)) {
      onInterestsChange(interestIds.filter((x) => x !== id));
    } else if (interestIds.length < MAX_INTERESTS) {
      onInterestsChange([...interestIds, id]);
    }
  };
  return (
    <View style={styles.wrap}>
      <TextField
        label="Quelques mots sur toi"
        value={bio}
        onChangeText={onBioChange}
        multiline
        maxLength={500}
        showCounter
        placeholder="Ce qui te fait vibrer, ce que tu cherches, ton meilleur plan du dimanche…"
        testID="profile-bio"
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Centres d’intérêt ({interestIds.length}/{MAX_INTERESTS})
        </Text>
        {interests.isLoading ? (
          <Skeleton height={120} />
        ) : (
          <View style={styles.chips}>
            {(interests.data ?? []).map((i) => (
              <Chip key={i.id} label={i.label} selected={interestIds.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
            ))}
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.lg },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
