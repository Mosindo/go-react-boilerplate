import React, { useState } from "react";
import { View } from "react-native";
import type { OwnProfile } from "../../api/types";
import { useInterests } from "../../hooks/queries";
import { Button, Chip, ErrorState, Input, Loader, Text, spacing } from "../../shared/ui";
import { MAX_BIO_LENGTH } from "../../utils/validation";

const MAX_INTERESTS = 10;

type Props = { profile: OwnProfile; submitLabel: string; busy?: boolean; onSubmit: (v: { bio: string; city: string; interests: string[] }) => void; showCity?: boolean };

export function AboutForm({ profile, submitLabel, busy, onSubmit, showCity = false }: Props) {
  const interests = useInterests();
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city);
  const [selected, setSelected] = useState<string[]>(profile.interests.map((i) => i.slug));

  const toggle = (slug: string) => setSelected((s) => (s.includes(slug) ? s.filter((x) => x !== slug) : s.length < MAX_INTERESTS ? [...s, slug] : s));

  if (interests.isPending) return <Loader />;
  if (interests.isError) return <ErrorState message="Impossible de charger les centres d'intérêt." onRetry={() => void interests.refetch()} />;

  return (
    <View style={{ gap: spacing.lg }}>
      <Input label="Bio" value={bio} onChangeText={setBio} multiline maxLength={MAX_BIO_LENGTH} hint={`${bio.length}/${MAX_BIO_LENGTH} — quelques mots sur vous`} testID="about-bio" />
      {showCity ? <Input label="Ville" value={city} onChangeText={setCity} maxLength={80} testID="about-city" /> : null}
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" tone="muted">Centres d'intérêt ({selected.length}/{MAX_INTERESTS})</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {interests.data.interests.map((i) => (
            <Chip key={i.slug} label={i.label} selected={selected.includes(i.slug)} onPress={() => toggle(i.slug)} testID={`interest-${i.slug}`} />
          ))}
        </View>
      </View>
      <Button title={submitLabel} onPress={() => onSubmit({ bio: bio.trim(), city: city.trim(), interests: selected })} loading={busy} testID="about-submit" />
    </View>
  );
}
