import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { safetyApi } from "../api/endpoints";
import { REPORT_REASONS, type ReportReason } from "../api/types";
import { errorMessage } from "../lib/errors";
import { useTheme } from "../theme";
import { Button, Dialog, Input, Text, useToast } from "../ui";

type Props = { userId: string; name: string; onDone: () => void };

/** Block + report dialogs shared by the profile and chat screens. */
export function useSafetyDialogs({ userId, name, onDone }: Props) {
  const toast = useToast();
  const [open, setOpen] = useState<"block" | "report" | null>(null);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const close = () => {
    setOpen(null);
    setReason(null);
    setDetails("");
  };

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      close();
      toast.show(`${name} est bloqué(e).`);
      onDone();
    },
    onError: (err) => toast.show(errorMessage(err), "error")
  });
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason ?? "other", details.trim(), alsoBlock),
    onSuccess: () => {
      const blocked = alsoBlock;
      close();
      toast.show("Merci, votre signalement a été envoyé.");
      if (blocked) onDone();
    },
    onError: (err) => toast.show(errorMessage(err), "error")
  });

  const t = useTheme();
  const dialogs = (
    <>
      <Dialog
        visible={open === "block"}
        title={`Bloquer ${name} ?`}
        message="Vous ne verrez plus ce profil et ne pourrez plus échanger. La personne n'est pas prévenue. Vous pourrez la débloquer depuis les paramètres."
        confirmLabel="Bloquer"
        confirmVariant="danger"
        loading={block.isPending}
        onConfirm={() => block.mutate()}
        onCancel={close}
      />
      <Dialog
        visible={open === "report"}
        title={`Signaler ${name}`}
        confirmLabel="Envoyer le signalement"
        loading={report.isPending}
        onConfirm={() => (reason ? report.mutate() : toast.show("Choisissez un motif.", "error"))}
        onCancel={close}
      >
        <View style={styles.reasons}>
          {REPORT_REASONS.map((r) => (
            <Pressable
              key={r.value}
              accessibilityRole="radio"
              accessibilityState={{ selected: reason === r.value }}
              onPress={() => setReason(r.value)}
              style={[
                styles.reason,
                {
                  borderColor: reason === r.value ? t.primary : t.border,
                  backgroundColor: reason === r.value ? t.primarySoft : "transparent"
                }
              ]}
            >
              <Text>{r.label}</Text>
            </Pressable>
          ))}
        </View>
        <Input
          label="Détails (facultatif)"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: alsoBlock }}
          onPress={() => setAlsoBlock((v) => !v)}
        >
          <Text variant="caption">{alsoBlock ? "☑" : "☐"} Bloquer aussi cette personne</Text>
        </Pressable>
      </Dialog>
    </>
  );
  return { dialogs, openBlock: () => setOpen("block"), openReport: () => setOpen("report") };
}

export function SafetyMenu(props: Props) {
  const { dialogs, openBlock, openReport } = useSafetyDialogs(props);
  return (
    <View style={styles.menu}>
      <Button label="Signaler" variant="ghost" onPress={openReport} />
      <Button label="Bloquer" variant="danger" onPress={openBlock} />
      {dialogs}
    </View>
  );
}

const styles = StyleSheet.create({
  menu: { marginTop: 16, gap: 8 },
  reasons: { gap: 8 },
  reason: { borderWidth: 1, borderRadius: 12, paddingHorizontal: 14, paddingVertical: 10 }
});
