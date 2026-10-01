import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import * as Location from "expo-location";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { photosApi, profileApi, type UploadFile } from "../api/endpoints";
import { keys } from "../api/keys";
import type { Gender, Preferences, SelfProfile } from "../api/types";
import { ageOn, splitIsoDate, toIsoDate } from "../lib/dates";
import { genderLabel, genderPlural } from "../lib/format";
import { MAX_AGE, MAX_INTERESTS, MIN_AGE, clamp, validateFirstName } from "../lib/validation";
import { radii, spacing, useTheme } from "../theme/theme";
import { AuthImage } from "./AuthImage";
import { useFeedback } from "./Feedback";
import { Button, Chip, ErrorText, Row, Text, TextField } from "./ui";

const GENDERS: Gender[] = ["woman", "man", "non_binary"];

export function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : "Une erreur est survenue.";
}

// ───────────────────────────── Basics ─────────────────────────────

export function BasicsForm({
  profile,
  submitLabel,
  onSaved
}: {
  profile: SelfProfile | undefined;
  submitLabel: string;
  onSaved: (p: SelfProfile) => void;
}) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const initial = profile?.birthDate ? splitIsoDate(profile.birthDate) : { day: "", month: "", year: "" };
  const [name, setName] = useState(profile?.firstName ?? "");
  const [day, setDay] = useState(initial.day);
  const [month, setMonth] = useState(initial.month);
  const [year, setYear] = useState(initial.year);
  const [gender, setGender] = useState<Gender | "">(profile?.gender ?? "");
  const [errors, setErrors] = useState<Record<string, string>>({});

  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (p) => {
      qc.setQueryData(keys.profile, p);
      onSaved(p);
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const submit = () => {
    const next: Record<string, string> = {};
    const nameError = validateFirstName(name);
    if (nameError) next.name = nameError;
    const iso = toIsoDate(day, month, year);
    if (!iso) next.birth = "Entrez une date valide (jour, mois, année).";
    else if (ageOn(iso) < MIN_AGE) next.birth = "Aurore est réservée aux personnes majeures (18 ans et plus).";
    if (!gender) next.gender = "Choisissez une option.";
    setErrors(next);
    if (Object.keys(next).length > 0 || !iso || !gender) return;
    save.mutate({
      firstName: name.trim(),
      birthDate: iso,
      gender,
      bio: profile?.bio ?? "",
      city: profile?.city ?? ""
    });
  };

  return (
    <View>
      <TextField
        label="Prénom"
        value={name}
        onChangeText={setName}
        error={errors.name}
        autoCapitalize="words"
        autoComplete="given-name"
        maxLength={40}
        testID="basics-name"
      />
      <Text variant="label" style={{ marginBottom: spacing.xs }}>
        Date de naissance
      </Text>
      <View style={styles.dateRow}>
        <TextField label="Jour" value={day} onChangeText={setDay} keyboardType="number-pad" maxLength={2} placeholder="JJ" style={styles.dateInput} testID="basics-day" />
        <TextField label="Mois" value={month} onChangeText={setMonth} keyboardType="number-pad" maxLength={2} placeholder="MM" style={styles.dateInput} testID="basics-month" />
        <TextField label="Année" value={year} onChangeText={setYear} keyboardType="number-pad" maxLength={4} placeholder="AAAA" style={[styles.dateInput, { minWidth: 96 }]} testID="basics-year" />
      </View>
      {errors.birth ? (
        <ErrorText style={{ marginTop: -spacing.sm, marginBottom: spacing.lg }}>{errors.birth}</ErrorText>
      ) : (
        <Text variant="caption" muted style={{ marginTop: -spacing.sm, marginBottom: spacing.lg }}>
          Votre âge est affiché, jamais votre date de naissance.
        </Text>
      )}
      <Text variant="label" style={{ marginBottom: spacing.sm }}>
        Je suis
      </Text>
      <Row style={{ marginBottom: errors.gender ? spacing.xs : spacing.xl }}>
        {GENDERS.map((g) => (
          <Chip key={g} label={genderLabel(g)} selected={gender === g} onPress={() => setGender(g)} testID={`gender-${g}`} />
        ))}
      </Row>
      {errors.gender ? (
        <ErrorText style={{ marginBottom: spacing.lg }}>{errors.gender}</ErrorText>
      ) : null}
      <Button label={submitLabel} onPress={submit} loading={save.isPending} testID="basics-submit" />
    </View>
  );
}

// ─────────────────────────── Preferences ──────────────────────────

function Stepper({
  label,
  value,
  onChange,
  min,
  max,
  step = 1,
  suffix = "",
  testID
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  testID?: string;
}) {
  const t = useTheme();
  const btn = (icon: "remove" | "add", delta: number, name: string) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${name} ${label}`}
      onPress={() => onChange(clamp(value + delta, min, max))}
      style={[styles.stepBtn, { borderColor: t.border, backgroundColor: t.surface }]}
    >
      <Ionicons name={icon} size={20} color={t.text} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" style={{ flex: 1 }}>
        {label}
      </Text>
      {btn("remove", -step, "Diminuer")}
      <Text variant="heading" style={{ minWidth: 72, textAlign: "center" }} testID={testID}>
        {value}
        {suffix}
      </Text>
      {btn("add", step, "Augmenter")}
    </View>
  );
}

export function PreferencesForm({
  profile,
  submitLabel,
  onSaved
}: {
  profile: SelfProfile;
  submitLabel: string;
  onSaved: (p: SelfProfile) => void;
}) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences.interestedIn as Gender[]);
  const [minAge, setMinAge] = useState(profile.preferences.minAge);
  const [maxAge, setMaxAge] = useState(profile.preferences.maxAge);
  const [distance, setDistance] = useState(profile.preferences.maxDistanceKm);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: (p: Preferences) => profileApi.savePreferences(p),
    onSuccess: (p) => {
      qc.setQueryData(keys.profile, p);
      void qc.invalidateQueries({ queryKey: ["discover"] });
      onSaved(p);
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const toggle = (g: Gender) =>
    setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]));

  const submit = () => {
    if (interestedIn.length === 0) return setError("Choisissez au moins une option.");
    setError(null);
    save.mutate({ interestedIn, minAge, maxAge, maxDistanceKm: distance });
  };

  return (
    <View>
      <Text variant="label" style={{ marginBottom: spacing.sm }}>
        Je souhaite rencontrer
      </Text>
      <Row style={{ marginBottom: spacing.sm }}>
        {GENDERS.map((g) => (
          <Chip key={g} label={genderPlural(g)} selected={interestedIn.includes(g)} onPress={() => toggle(g)} testID={`interested-${g}`} />
        ))}
      </Row>
      {error ? (
        <ErrorText style={{ marginBottom: spacing.md }}>{error}</ErrorText>
      ) : null}
      <View style={{ height: spacing.lg }} />
      <Stepper label="Âge minimum" value={minAge} min={MIN_AGE} max={maxAge} onChange={setMinAge} testID="pref-min-age" />
      <Stepper label="Âge maximum" value={maxAge} min={minAge} max={MAX_AGE} onChange={setMaxAge} testID="pref-max-age" />
      <Stepper label="Distance maximale" value={distance} min={5} max={500} step={5} suffix=" km" onChange={setDistance} testID="pref-distance" />
      <Text variant="caption" muted style={{ marginBottom: spacing.xl }}>
        La distance ne s&apos;applique que si vous partagez votre position approximative.
      </Text>
      <Button label={submitLabel} onPress={submit} loading={save.isPending} testID="prefs-submit" />
    </View>
  );
}

// ───────────────────────────── Photos ─────────────────────────────

async function pickImage(): Promise<UploadFile | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 0.85, // < 1 makes the OS re-encode, which also bakes in EXIF orientation
    allowsEditing: false
  });
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  const name = asset.fileName ?? "photo.jpg";
  const type = asset.mimeType ?? "image/jpeg";
  const file = (asset as { file?: Blob }).file;
  if (file) return { uri: asset.uri, name, type, blob: file };
  if (typeof document !== "undefined") {
    const blob = await (await fetch(asset.uri)).blob();
    return { uri: asset.uri, name, type, blob };
  }
  return { uri: asset.uri, name, type };
}

export function PhotoManager({ profile }: { profile: SelfProfile }) {
  const t = useTheme();
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const photos = profile.photos;
  const MAX = 6;

  const refresh = () => qc.invalidateQueries({ queryKey: keys.profile });
  const onError = (e: unknown) => toast(errorMessage(e), "error");

  const add = useMutation({ mutationFn: photosApi.add, onSuccess: refresh, onError });
  const replace = useMutation({
    mutationFn: ({ id, file }: { id: string; file: UploadFile }) => photosApi.replace(id, file),
    onSuccess: refresh,
    onError
  });
  const remove = useMutation({ mutationFn: photosApi.remove, onSuccess: refresh, onError });
  const reorder = useMutation({ mutationFn: photosApi.reorder, onSuccess: refresh, onError });

  const choose = async (then: (f: UploadFile) => void) => {
    try {
      const file = await pickImage();
      if (file) then(file);
    } catch (e) {
      onError(e);
    }
  };

  const move = (index: number, to: number) => {
    const ids = photos.map((p) => p.id);
    const [moved] = ids.splice(index, 1);
    if (!moved) return;
    ids.splice(to, 0, moved);
    reorder.mutate(ids);
  };

  const askRemove = async (id: string) => {
    const ok = await confirm({ title: "Supprimer cette photo ?", confirmLabel: "Supprimer", destructive: true });
    if (ok) remove.mutate(id);
  };

  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: MAX }).map((_, i) => {
          const photo = photos[i];
          if (!photo) {
            const canAdd = i === photos.length;
            return (
              <Pressable
                key={`slot-${i}`}
                accessibilityRole="button"
                accessibilityLabel="Ajouter une photo"
                disabled={!canAdd || busy}
                onPress={() => void choose((f) => add.mutate(f))}
                testID={canAdd ? "photo-add" : undefined}
                style={[styles.slot, { borderColor: t.border, backgroundColor: t.surface, opacity: canAdd ? 1 : 0.45 }]}
              >
                <Ionicons name="add" size={28} color={t.primary} />
              </Pressable>
            );
          }
          return (
            <View key={photo.id} style={[styles.slot, { borderColor: i === 0 ? t.primary : t.border }]} testID={`photo-${i}`}>
              <AuthImage path={photo.thumbUrl} label={`Photo ${i + 1}`} style={StyleSheet.absoluteFill as object} />
              {i === 0 ? (
                <View style={[styles.mainTag, { backgroundColor: t.primary }]}>
                  <Text variant="caption" color={t.onPrimary} style={{ fontWeight: "700" }}>
                    Principale
                  </Text>
                </View>
              ) : null}
              <View style={styles.photoActions}>
                {i > 0 ? (
                  <PhotoAction icon="arrow-back" label="Déplacer vers l'avant" onPress={() => move(i, i - 1)} />
                ) : null}
                {i < photos.length - 1 ? (
                  <PhotoAction icon="arrow-forward" label="Déplacer vers l'arrière" onPress={() => move(i, i + 1)} />
                ) : null}
                <PhotoAction icon="swap-horizontal" label="Remplacer la photo" onPress={() => void choose((f) => replace.mutate({ id: photo.id, file: f }))} />
                <PhotoAction icon="trash" label="Supprimer la photo" onPress={() => void askRemove(photo.id)} testID={`photo-delete-${i}`} />
              </View>
            </View>
          );
        })}
      </View>
      <Text variant="caption" muted style={{ marginTop: spacing.md }}>
        {photos.length}/{MAX} photos · la première est votre photo principale · JPEG, PNG ou WebP, 8 Mo maximum.
      </Text>
    </View>
  );
}

function PhotoAction({
  icon,
  label,
  onPress,
  testID
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  testID?: string;
}) {
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} onPress={onPress} testID={testID} hitSlop={4} style={styles.photoAction}>
      <Ionicons name={icon} size={14} color="#FFFFFF" />
    </Pressable>
  );
}

// ──────────────────────────── Interests ───────────────────────────

export function InterestPicker({
  selected,
  onChange
}: {
  selected: number[];
  onChange: (ids: number[]) => void;
}) {
  const { toast } = useFeedback();
  const { data, isLoading } = useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: Infinity });
  if (isLoading || !data) return <Text muted>Chargement…</Text>;
  const toggle = (id: number) => {
    if (selected.includes(id)) return onChange(selected.filter((x) => x !== id));
    if (selected.length >= MAX_INTERESTS) return toast(`${MAX_INTERESTS} centres d'intérêt maximum.`);
    onChange([...selected, id]);
  };
  return (
    <View>
      <Row>
        {data.map((i) => (
          <Chip key={i.id} label={i.label} selected={selected.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
        ))}
      </Row>
      <Text variant="caption" muted style={{ marginTop: spacing.md }}>
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}

// ──────────────────────────── Location ────────────────────────────

/** Asks the OS for the position once, sends it to the API (which stores only a ~1 km grid cell). */
export function ShareLocationButton({
  onDone,
  variant = "secondary"
}: {
  onDone?: (p: SelfProfile) => void;
  variant?: "primary" | "secondary";
}) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const [busy, setBusy] = useState(false);
  const run = async () => {
    setBusy(true);
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (perm.status !== "granted") {
        toast("Position refusée : vous pouvez l'autoriser dans les réglages de votre appareil.", "error");
        return;
      }
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city: string | undefined;
      try {
        const [place] = await Location.reverseGeocodeAsync(pos.coords);
        city = place?.city ?? place?.subregion ?? undefined;
      } catch {
        city = undefined; // reverse geocoding is a nicety, never a blocker
      }
      const p = await profileApi.saveLocation(pos.coords.latitude, pos.coords.longitude, city);
      qc.setQueryData(keys.profile, p);
      void qc.invalidateQueries({ queryKey: ["discover"] });
      toast("Position mise à jour (approximative).", "success");
      onDone?.(p);
    } catch (e) {
      toast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };
  return <Button label="Partager ma position approximative" icon="location" variant={variant} loading={busy} onPress={() => void run()} testID="share-location" />;
}

export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return v;
}

const styles = StyleSheet.create({
  dateRow: { flexDirection: "row", gap: spacing.md },
  dateInput: { textAlign: "center" },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.md, marginBottom: spacing.lg },
  stepBtn: {
    width: 44,
    height: 44,
    borderRadius: 22,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  slot: {
    width: "30.5%",
    aspectRatio: 3 / 4,
    borderRadius: radii.md,
    borderWidth: 1.5,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  mainTag: { position: "absolute", top: 6, left: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  photoActions: {
    position: "absolute",
    bottom: 4,
    left: 4,
    right: 4,
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "flex-end",
    gap: 4
  },
  photoAction: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: "rgba(0,0,0,0.6)",
    alignItems: "center",
    justifyContent: "center"
  }
});
