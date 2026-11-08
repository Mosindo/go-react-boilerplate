import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../../api/endpoints";
import type { Gender, MyProfile } from "../../api/types";
import { keys, useInterests, useMyProfile } from "../../hooks/queries";
import { geocodeCity, locateMe } from "../../lib/device";
import type { RootParamList } from "../../navigation/types";
import { spacing } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { ErrorState, LoadingView } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";
import { useToast } from "../../ui/Toast";
import { PhotoManager } from "./PhotoManager";
import { GenderChips, Section } from "./shared";

type Props = NativeStackScreenProps<RootParamList, "EditProfile">;

function Form({ profile, navigation }: { profile: MyProfile; navigation: Props["navigation"] }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender | undefined>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [picked, setPicked] = useState<string[]>(profile.interests.map((i) => i.slug));
  const [city, setCity] = useState(profile.city);

  const save = useMutation({
    mutationFn: () =>
      profileApi.update({ firstName: firstName.trim(), gender, bio: bio.trim(), interests: picked }),
    onSuccess: (updated) => {
      queryClient.setQueryData(keys.profile, updated);
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      toast.show("Profil enregistré.", "success");
      navigation.goBack();
    },
    onError: (e) => toast.show(e.message, "error")
  });

  const locate = useMutation({
    mutationFn: async (manual: boolean) => {
      const where = manual ? await geocodeCity(city) : await locateMe();
      return profileApi.setLocation(where.latitude, where.longitude, where.city ?? (city || undefined));
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(keys.profile, updated);
      setCity(updated.city);
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      toast.show("Position mise à jour.", "success");
    },
    onError: (e) => toast.show(e.message, "error")
  });

  const toggle = (slug: string) =>
    setPicked((cur) =>
      cur.includes(slug) ? cur.filter((s) => s !== slug) : cur.length >= 8 ? cur : [...cur, slug]
    );

  return (
    <Screen scroll>
      <Section title="Photos">
        <PhotoManager photos={profile.photos} />
      </Section>
      <Section title="À propos de toi">
        <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
        <Text variant="label" tone="textMuted" style={styles.gap}>
          Genre
        </Text>
        <GenderChips value={gender ? [gender] : []} onToggle={setGender} />
        <TextField
          label="Bio"
          value={bio}
          onChangeText={setBio}
          multiline
          maxLength={500}
          helper={`${bio.length}/500`}
          testID="edit-bio"
        />
      </Section>
      <Section title={`Centres d'intérêt (${picked.length}/8)`}>
        <View style={styles.chips}>
          {(interests.data ?? []).map((i) => (
            <Chip
              key={i.slug}
              label={i.label}
              selected={picked.includes(i.slug)}
              onPress={() => toggle(i.slug)}
            />
          ))}
        </View>
      </Section>
      <Section title="Localisation">
        <TextField
          label="Ville"
          value={city}
          onChangeText={setCity}
          helper="Seule ta ville et une distance approximative sont visibles."
        />
        <View style={styles.row}>
          <Button
            title="Utiliser ma position"
            variant="secondary"
            icon="locate"
            onPress={() => locate.mutate(false)}
            loading={locate.isPending}
            style={styles.flex}
          />
          <Button
            title="Chercher cette ville"
            variant="secondary"
            onPress={() => locate.mutate(true)}
            disabled={!city.trim() || locate.isPending}
            style={styles.flex}
          />
        </View>
      </Section>
      <Button
        title="Enregistrer"
        onPress={() => save.mutate()}
        loading={save.isPending}
        disabled={!firstName.trim()}
        testID="edit-save"
      />
    </Screen>
  );
}

export function EditProfileScreen({ navigation }: Props) {
  const query = useMyProfile();
  const [ready, setReady] = useState(false);
  useEffect(() => setReady(true), []);
  if (query.isPending || !ready)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );
  return <Form profile={query.data} navigation={navigation} />;
}

const styles = StyleSheet.create({
  gap: { marginBottom: spacing.xxs },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  row: { flexDirection: "row", gap: spacing.xs },
  flex: { flex: 1 }
});
