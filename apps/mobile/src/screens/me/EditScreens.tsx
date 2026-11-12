import React, { useState } from "react";
import { Platform, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi, profileApi, safetyApi } from "../../api/endpoints";
import type { OwnProfile, Preferences, RelationshipGoal } from "../../api/types";
import { InterestPicker } from "../../components/InterestPicker";
import { getCoarseLocation } from "../../components/location";
import { PhotoGrid } from "../../components/PhotoGrid";
import { PreferencesForm } from "../../components/PreferencesForm";
import { useAuth } from "../../hooks/useAuth";
import { useBlocks, useInterests, useInvalidateRelationships, useOwnProfile, useSetProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, Button, Chip, EmptyState, Header, InlineError, ListRow, Loading, Screen, Section, SwitchRow, Text, TextField, showToast } from "../../ui";
import { goalLabels, validatePassword } from "../../utils/format";

type Props<K extends keyof AppStackParamList> = NativeStackScreenProps<AppStackParamList, K>;

function useSaver() {
  const setProfile = useSetProfile();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const save = async (action: () => Promise<OwnProfile>, message = "Modifications enregistrées") => {
    setBusy(true);
    setError(null);
    try {
      setProfile(await action());
      showToast(message, "success");
      return true;
    } catch (e) {
      setError(errorMessage(e));
      return false;
    } finally {
      setBusy(false);
    }
  };
  return { busy, error, save };
}

function Shell({ title, onBack, children }: { title: string; onBack: () => void; children: React.ReactNode }) {
  return (
    <Screen scroll keyboard padded={false}>
      <Header title={title} onBack={onBack} />
      <View style={styles.body}>{children}</View>
    </Screen>
  );
}

export function EditProfileScreen({ navigation }: Props<"EditProfile">) {
  const { data: p } = useOwnProfile();
  const { busy, error, save } = useSaver();
  const [firstName, setFirstName] = useState(p?.firstName ?? "");
  const [bio, setBio] = useState(p?.bio ?? "");
  const [occupation, setOccupation] = useState(p?.occupation ?? "");
  const [city, setCity] = useState(p?.city ?? "");
  const [goal, setGoal] = useState<RelationshipGoal>(p?.relationshipGoal ?? "unsure");
  if (!p) return <Loading />;
  return (
    <Shell title="Informations" onBack={navigation.goBack}>
      <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
      <TextField label="Bio" value={bio} onChangeText={setBio} multiline maxLength={500} counter />
      <TextField label="Métier" value={occupation} onChangeText={setOccupation} maxLength={80} />
      <TextField label="Ville" value={city} onChangeText={setCity} maxLength={80} />
      <Text variant="label" tone="muted">
        Je cherche plutôt
      </Text>
      <View style={styles.chips}>
        {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
          <Chip key={g} label={goalLabels[g]} selected={goal === g} onPress={() => setGoal(g)} />
        ))}
      </View>
      <Text variant="caption" tone="subtle">
        Date de naissance : {p.birthdate.split("-").reverse().join("/")} (non modifiable, jamais affichée).
      </Text>
      <InlineError message={error} />
      <Button
        label="Enregistrer"
        loading={busy}
        onPress={async () => {
          const ok = await save(() =>
            profileApi.upsert({ firstName, birthdate: p.birthdate, gender: p.gender, bio, occupation, relationshipGoal: goal, city })
          );
          if (ok) navigation.goBack();
        }}
      />
    </Shell>
  );
}

export function EditPhotosScreen({ navigation }: Props<"EditPhotos">) {
  const { data: p } = useOwnProfile();
  const setProfile = useSetProfile();
  if (!p) return <Loading />;
  return (
    <Shell title="Photos" onBack={navigation.goBack}>
      <Text tone="muted">
        Touchez une photo pour la remplacer, la supprimer ou en faire votre photo principale. Les métadonnées (dont la position GPS) sont supprimées à l’envoi.
      </Text>
      <PhotoGrid
        photos={p.photos}
        onChange={(photos) => setProfile({ ...p, photos, isComplete: photos.length > 0 && p.preferences.interestedIn.length > 0 })}
      />
    </Shell>
  );
}

export function EditPreferencesScreen({ navigation }: Props<"EditPreferences">) {
  const { data: p } = useOwnProfile();
  const { busy, error, save } = useSaver();
  const [prefs, setPrefs] = useState<Preferences | null>(p?.preferences ?? null);
  if (!p || !prefs) return <Loading />;
  return (
    <Shell title="Préférences" onBack={navigation.goBack}>
      <PreferencesForm value={prefs} onChange={setPrefs} />
      <Text variant="caption" tone="subtle">
        Les critères sont réciproques : vous ne voyez que des personnes dont les critères vous incluent aussi.
      </Text>
      <InlineError message={error} />
      <Button
        label="Enregistrer"
        loading={busy}
        disabled={!prefs.interestedIn.length}
        onPress={async () => (await save(() => profileApi.preferences(prefs))) && navigation.goBack()}
      />
    </Shell>
  );
}

export function EditInterestsScreen({ navigation }: Props<"EditInterests">) {
  const { data: p } = useOwnProfile();
  const catalogue = useInterests();
  const { busy, error, save } = useSaver();
  const [selected, setSelected] = useState<number[]>(p?.interests.map((i) => i.id) ?? []);
  if (!p || !catalogue.data) return <Loading />;
  return (
    <Shell title="Centres d'intérêt" onBack={navigation.goBack}>
      <InterestPicker
        catalogue={catalogue.data}
        selected={selected}
        onToggle={(id) => setSelected((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))}
      />
      <InlineError message={error} />
      <Button label="Enregistrer" loading={busy} onPress={async () => (await save(() => profileApi.interests(selected))) && navigation.goBack()} />
    </Shell>
  );
}

export function PrivacyScreen({ navigation }: Props<"Privacy">) {
  const { data: p } = useOwnProfile();
  const { busy, error, save } = useSaver();
  if (!p) return <Loading />;
  const setPrivacy = (patch: Partial<OwnProfile["privacy"]>) => void save(() => profileApi.privacy({ ...p.privacy, ...patch }));
  const updateLocation = async () => {
    const loc = await getCoarseLocation();
    if ("error" in loc) {
      showToast(loc.error, "error");
      return;
    }
    await save(() => profileApi.location(loc.latitude, loc.longitude, loc.city), "Position approximative mise à jour");
  };
  return (
    <Shell title="Confidentialité" onBack={navigation.goBack}>
      <Section title="Visibilité">
        <SwitchRow
          title="Apparaître dans la découverte"
          subtitle="Désactivez pour mettre votre profil en pause. Vos matchs restent."
          value={p.privacy.discoverable}
          onValueChange={(v) => setPrivacy({ discoverable: v })}
          disabled={busy}
        />
        <SwitchRow title="Afficher mon âge" value={p.privacy.showAge} onValueChange={(v) => setPrivacy({ showAge: v })} disabled={busy} />
        <SwitchRow
          title="Afficher la distance"
          subtitle="Toujours approximative, jamais votre position."
          value={p.privacy.showDistance}
          onValueChange={(v) => setPrivacy({ showDistance: v })}
          disabled={busy}
          last
        />
      </Section>
      <Section title="Localisation" footer="Votre position est arrondie à environ 1 km et n'est mise à jour que lorsque vous le demandez.">
        <ListRow
          icon="navigate-outline"
          title={p.hasLocation ? "Mettre à jour ma position" : "Activer la localisation"}
          subtitle={
            p.hasLocation && p.locationUpdatedAt
              ? `Dernière mise à jour : ${new Date(p.locationUpdatedAt).toLocaleDateString("fr-FR")}`
              : "Pour voir des personnes près de vous"
          }
          onPress={() => void updateLocation()}
        />
        {p.hasLocation ? (
          <ListRow
            icon="trash-outline"
            title="Supprimer ma position"
            destructive
            onPress={() => void save(() => profileApi.clearLocation(), "Position supprimée")}
            last
          />
        ) : null}
      </Section>
      <InlineError message={error} />
    </Shell>
  );
}

export function BlockedScreen({ navigation }: Props<"Blocked">) {
  const blocks = useBlocks();
  const invalidate = useInvalidateRelationships();
  if (blocks.isLoading) return <Loading />;
  return (
    <Shell title="Personnes bloquées" onBack={navigation.goBack}>
      {!blocks.data?.length ? (
        <EmptyState icon="shield-checkmark-outline" title="Personne n'est bloqué" message="Les personnes que vous bloquez ou signalez apparaissent ici." />
      ) : (
        <Section>
          {blocks.data.map((b, i) => (
            <ListRow
              key={b.userId}
              title={b.firstName}
              subtitle={`Bloqué·e le ${new Date(b.blockedAt).toLocaleDateString("fr-FR")}`}
              last={i === (blocks.data?.length ?? 0) - 1}
              right={
                <View style={styles.row}>
                  <Avatar url={b.photoUrl} size={32} />
                  <Button
                    label="Débloquer"
                    compact
                    variant="secondary"
                    onPress={async () => {
                      await safetyApi.unblock(b.userId).catch(() => undefined);
                      invalidate();
                    }}
                  />
                </View>
              }
            />
          ))}
        </Section>
      )}
    </Shell>
  );
}

export function AccountScreen({ navigation }: Props<"Account">) {
  const { signOut } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [deletePassword, setDeletePassword] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [busy, setBusy] = useState<"password" | "delete" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const changePassword = async () => {
    const invalid = validatePassword(next);
    if (invalid) return setError(invalid);
    setBusy("password");
    setError(null);
    try {
      await authApi.changePassword(current, next);
      setCurrent("");
      setNext("");
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(null);
    }
  };

  const deleteAccount = async () => {
    setBusy("delete");
    setError(null);
    try {
      await authApi.deleteAccount(deletePassword);
      showToast("Votre compte et vos données ont été supprimés.", "success");
      await signOut({ remote: false });
    } catch (e) {
      setError(errorMessage(e));
      setBusy(null);
    }
  };

  return (
    <Shell title="Compte" onBack={navigation.goBack}>
      <Text variant="heading">Changer de mot de passe</Text>
      <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry autoComplete="password" />
      <TextField
        label="Nouveau mot de passe"
        value={next}
        onChangeText={setNext}
        secureTextEntry
        autoComplete="new-password"
        hint="8 caractères minimum, avec une lettre et un chiffre."
      />
      <Button label="Mettre à jour" variant="secondary" loading={busy === "password"} disabled={!current || !next} onPress={changePassword} />

      <View style={styles.danger}>
        <Text variant="heading">Supprimer mon compte</Text>
        <Text tone="muted">
          Suppression immédiate et définitive : profil, photos, matchs, conversations et préférences. Les signalements que vous avez faits sont conservés de
          façon anonyme pour la modération.
        </Text>
        {!confirmDelete ? (
          <Button label="Supprimer mon compte" variant="danger" onPress={() => setConfirmDelete(true)} testID="account-delete" />
        ) : (
          <>
            <TextField
              label="Confirmez avec votre mot de passe"
              value={deletePassword}
              onChangeText={setDeletePassword}
              secureTextEntry
              testID="account-delete-password"
            />
            <Button
              label="Supprimer définitivement"
              variant="danger"
              loading={busy === "delete"}
              disabled={!deletePassword}
              onPress={deleteAccount}
              testID="account-delete-confirm"
            />
            <Button label="Annuler" variant="ghost" onPress={() => setConfirmDelete(false)} />
          </>
        )}
      </View>
      <InlineError message={error} />
      {Platform.OS === "web" ? null : <View style={styles.bottomSpace} />}
    </Shell>
  );
}

const styles = StyleSheet.create({
  body: { paddingHorizontal: spacing.lg, gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  danger: { gap: spacing.sm, marginTop: spacing.xl },
  bottomSpace: { height: spacing.xl }
});
