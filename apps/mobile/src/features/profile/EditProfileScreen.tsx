import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { Gender } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { isValidFirstName } from "../../lib/validation";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, ErrorState, Header, InlineError, Loading, Screen, Section, Text, TextField, toast } from "../../ui";
import { GenderPicker, InterestPicker } from "./fields";
import { profileToUpdate, useClearLocation, useInterests, useMyProfile, useUpdateLocation, useUpdateProfile } from "./hooks";
import { currentApproximateLocation } from "./media";
import { PhotoGrid } from "./PhotoGrid";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export function EditProfileScreen({ navigation }: Props) {
  const me = useMyProfile();
  const interests = useInterests();
  const update = useUpdateProfile();
  const updateLocation = useUpdateLocation();
  const clearLocation = useClearLocation();
  const [form, setForm] = useState(() => (me.data ? profileToUpdate(me.data.profile) : null));
  const [height, setHeight] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);

  useEffect(() => {
    if (me.data && !form) setForm(profileToUpdate(me.data.profile));
    if (me.data?.profile.heightCm && !height) setHeight(String(me.data.profile.heightCm));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me.data]);

  if (me.isPending || !form) return <Loading />;
  if (me.isError) return <ErrorState message={errorMessage(me.error)} />;
  const profile = me.data.profile;
  const set = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) => setForm({ ...form, [key]: value });

  const save = async () => {
    if (!isValidFirstName(form.firstName)) {
      setError("Prénom invalide.");
      return;
    }
    const heightCm = height.trim() ? Number(height) : null;
    if (heightCm !== null && (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230)) {
      setError("La taille doit être comprise entre 120 et 230 cm.");
      return;
    }
    setError(null);
    try {
      await update.mutateAsync({ ...form, firstName: form.firstName.trim(), heightCm });
      toast("Profil enregistré", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const locate = async () => {
    setLocating(true);
    const result = await currentApproximateLocation();
    setLocating(false);
    if ("error" in result) {
      toast(result.error, "error");
      return;
    }
    try {
      const updated = await updateLocation.mutateAsync({ ...result, city: result.city || form.city });
      setForm({ ...form, city: updated.profile.city });
      toast("Position approximative mise à jour", "success");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  return (
    <Screen
      scroll
      testID="edit-profile-screen"
      edges={["top", "bottom"]}
      footer={
        <View style={{ gap: spacing.xs }}>
          <InlineError message={error} />
          <Button label="Enregistrer" onPress={save} loading={update.isPending} testID="edit-profile-save" />
        </View>
      }
    >
      <Header title="Modifier mon profil" onBack={() => navigation.goBack()} />
      <Section title="Photos">
        <View style={styles.padded}>
          <PhotoGrid photos={profile.photos} />
        </View>
      </Section>
      <View style={styles.form}>
        <TextField label="Prénom" value={form.firstName} onChangeText={(v) => set("firstName", v)} maxLength={40} />
        <View style={{ gap: spacing.xs }}>
          <Text variant="label" color="textMuted">
            Genre
          </Text>
          <GenderPicker value={form.gender} onChange={(g: Gender) => set("gender", g)} />
        </View>
        <TextField
          label="Bio"
          value={form.bio}
          onChangeText={(v) => set("bio", v)}
          multiline
          maxLength={500}
          hint={`${form.bio.length}/500`}
          testID="edit-bio"
        />
        <TextField label="Métier" value={form.jobTitle} onChangeText={(v) => set("jobTitle", v)} maxLength={60} />
        <TextField
          label="Taille (cm)"
          value={height}
          onChangeText={(v) => setHeight(v.replace(/\D/g, "").slice(0, 3))}
          keyboardType="number-pad"
          placeholder="Facultatif"
        />
        <TextField label="Ville" value={form.city} onChangeText={(v) => set("city", v)} maxLength={80} />
        <View style={styles.locationRow}>
          <Button
            label={profile.hasLocation ? "Actualiser ma position" : "Utiliser ma position"}
            icon="navigate-outline"
            variant="secondary"
            compact
            loading={locating}
            onPress={locate}
          />
          {profile.hasLocation ? (
            <Button
              label="Oublier ma position"
              variant="ghost"
              compact
              onPress={() => clearLocation.mutate(undefined, { onSuccess: () => toast("Position supprimée") })}
            />
          ) : null}
        </View>
        <Text variant="caption" color="textSubtle">
          Date de naissance : {profile.birthdate ?? "—"} (non modifiable). Seul votre âge est visible.
        </Text>
        {interests.data ? (
          <View style={{ gap: spacing.xs }}>
            <Text variant="label" color="textMuted">
              Centres d’intérêt
            </Text>
            <InterestPicker all={interests.data} selected={form.interestIds} onChange={(ids) => set("interestIds", ids)} />
          </View>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  padded: { paddingVertical: spacing.sm },
  form: { gap: spacing.lg },
  locationRow: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: -spacing.sm }
});
