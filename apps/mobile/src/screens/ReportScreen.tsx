import { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { moderationApi } from "../api";
import type { ReportReason } from "../api/types";
import { useToast } from "../components/feedback";
import { Button, Chip, Screen, Text, TextField } from "../components/ui";
import { errorMessage } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { spacing } from "../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Report">;

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "spam", label: "Spam ou arnaque" },
  { value: "harassment", label: "Harcèlement" },
  { value: "fake", label: "Faux profil" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" },
];

export function ReportScreen({ navigation, route }: Props) {
  const { userId, name } = route.params;
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(route.params.defaultReason ?? null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(false);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) {
      return;
    }
    setBusy(true);
    try {
      await moderationApi.report(userId, reason, details.trim());
      if (alsoBlock) {
        await moderationApi.block(userId);
      }
      toast.show("Merci, votre signalement a bien été transmis.", "success");
      if (alsoBlock) {
        navigation.popToTop();
      } else {
        navigation.goBack();
      }
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text style={{ marginBottom: spacing.md }} variant="heading">
        Signaler {name}
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", marginBottom: spacing.md }}>
        {REASONS.map((r) => (
          <Chip key={r.value} label={r.label} onPress={() => setReason(r.value)} selected={reason === r.value} />
        ))}
      </View>
      <TextField
        label="Détails (facultatif)"
        maxLength={1000}
        multiline
        onChangeText={setDetails}
        style={{ minHeight: 96, textAlignVertical: "top" }}
        value={details}
      />
      <View style={{ flexDirection: "row", marginBottom: spacing.lg }}>
        <Chip label="Bloquer aussi cette personne" onPress={() => setAlsoBlock(!alsoBlock)} selected={alsoBlock} />
      </View>
      <Button
        disabled={!reason}
        loading={busy}
        onPress={() => void submit()}
        title="Envoyer le signalement"
        variant="danger"
      />
    </Screen>
  );
}
