import { useAuth } from "@/auth/AuthContext";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const tiles = [
  { title: "Arbeiter", desc: "Mitarbeiter verwalten" },
  { title: "Baustellen", desc: "Einsatzorte verwalten" },
  { title: "Arbeitszeiten", desc: "Alle Zeitbuchungen + Summen" },
  { title: "Urlaubsanträge", desc: "Genehmigen / ablehnen" },
  { title: "Überstunden", desc: "Saldo je Arbeiter/Monat" },
  { title: "Berichte & Dokumente", desc: "Monatsbericht, Lohnzettel" },
];

export function DashboardPage() {
  const { arbeiter } = useAuth();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Willkommen, {arbeiter?.name}</h1>
        <p className="text-muted-foreground">Übersicht — die Screens folgen in Part 2.</p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {tiles.map((t) => (
          <Card key={t.title} className="opacity-70">
            <CardHeader>
              <CardTitle className="text-base">{t.title}</CardTitle>
              <CardDescription>{t.desc}</CardDescription>
            </CardHeader>
            <CardContent className="text-xs text-muted-foreground">Bald verfügbar</CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
