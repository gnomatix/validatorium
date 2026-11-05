# Micropublications ontology status and related claim-evidence models

**Status: UNVERIFIED.** Extracted on 2026-11-05 from a deep-research run that was stopped
during its fetch phase, before adversarial verification and synthesis. Each line is a
claim as extracted from one fetched source by a reader agent; none has been checked
against a second source. Treat as leads to confirm, not as findings.

Fetched sources with extracted claims: 0; claims: 45

## Claims

- The timclark001/mp GitHub repository is the author's own publication of the Micropublications ontology (mp.owl) and has had no commits since 13 May 2016; its entire history is three commits on that single day, indicating no active maintenance of the repository since.  
  Fetch label: fetch:github.com
- The mp.owl file in the repository declares ontology IRI http://purl.org/mp/ with owl:versionInfo "1.18" and owl:priorVersion "1.17" (the header comment dates version 1.17 to May 12, 2016), so v1.18 is the latest and final published version of the ontology.  
  Fetch label: fetch:github.com
- mp.owl v1.18 declares no owl:imports; it reuses PROV terms directly (declaring prov:Agent, prov:Entity, prov:Organization, prov:Person) and uses dct:bibliographicCitation, but does not import or reference Open Annotation, SWAN, PAV, FOAF, SIO or the nanopublication vocabulary.  
  Fetch label: fetch:github.com
- In mp.owl, support and challenge are modeled as object properties between statements/representations (:supports, :challenges), not as classes, and the model is described as a multi-polar (+,-,0) argument graph; the ontology has 19 classes in the mp namespace including Claim, Micropublication, Statement, Representation, Sentence, Method, Procedure, Material, Data, Attribution, Artifact, Reference, Qualifier and SemanticQualifier.  
  Fetch label: fetch:github.com
- The repository has minimal community uptake: 1 star, 0 forks, 1 watcher, no releases, no packages, MIT license, and only three files (LICENSE, README.md, mp.owl); the README cites only the 2014 Journal of Biomedical Semantics paper and mentions no software implementations, tooling or registries.  
  Fetch label: fetch:github.com
- Micropublications was applied (by its own authors Ciccarese and Clark, with Schneider and Boyce) as an early real-world use case in 2014: a proposed redesign of the Drug Interaction Knowledge Base (DIKB v1.2) combining MP with the Open Annotation Data Model, with the ontology identified by http://purl.org/mp/ and cited as Clark, Ciccarese & Goble 2014, J Biomed Semantics 5:28.  
  Fetch label: fetch:ceur-ws.org
- The paper describes MP's core design for claim-evidence representation: a mp:Micropublication mp:argues a mp:Claim; Data, Methods and Materials are mp:Representations; and any Representation may mp:support or mp:challenge any other Representation, with support/challenge chains transitively closable across the literature (i.e., support/challenge is an evidential relation between representations, not a vote).  
  Fetch label: fetch:ceur-ws.org
- The prior DIKB 1.2 used the SWAN discourse ontology (swanco:citesAsSupportingEvidence / swanco:citesAsRefutingEvidence) to link assertions to whole source documents; MP plus OA was adopted as a successor to SWAN for finer-grained linkage from claims to quoted data/methods/materials in source documents, with claims grounded by qualifiers reusing external ontology identifiers (DRON drug IDs, Protein Ontology IDs) and the DIKB evidence taxonomy.  
  Fetch label: fetch:ceur-ws.org
- As of 2014, MP was described by its authors as a 'relatively new ontology' that had not yet been used to represent both unstructured claims and corresponding logical-formalism sentences, which was an open modeling challenge for the DIKB use case.  
  Fetch label: fetch:ceur-ws.org
- Tooling for MP in 2014 consisted of a Micropublication plugin for the Domeo annotation toolkit, which the authors were modifying to support evidence collection for the NLM-funded PDDI project (grant 1R01LM011838-01); the paper does not mention nanopublications, PROV-O or SEPIO.  
  Fetch label: fetch:ceur-ws.org
- The DIKB-Micropublication repository (University of Pittsburgh DBMI, PI Richard Boyce) is a software implementation that uses the Micropublications ontology together with the Open Annotation Data Model and DIDEO to represent drug-drug interaction claims and evidence, i.e. a concrete third-party use of the Micropublications model.  
  Fetch label: fetch:github.com
- This repository hosts a copy of the Micropublications ontology file at data/mp_1.18.owl, committed by Jodi Schneider in May 2016 as a rename/update of mp_1.17_updated_with_formalizes.owl, with the file's internal version metadata dated August 10, 2015 — meaning mp.owl v1.18 was produced within the DDI evidence project, after the 2014 JBMS paper.  
  Fetch label: fetch:github.com
- The repository is effectively dormant: the three most recent commits are README-only edits on 2017-08-09 and 2020-01-22, and the last substantive content commits date from December 2015 (ER diagram for MP RDB schema) and May 2016 (mp_1.18.owl); the repo has 7 stars, 1 fork, 344 commits and 16 open issues.  
  Fetch label: fetch:github.com
- The Micropublications ontology authors (Tim Clark, Paolo Ciccarese) were formal collaborators on this DDI knowledge-base project, alongside Paul Groth (nanopublications) and Jodi Schneider, funded by NLM grant 1R01LM011838-01; the project also studied nanopublication guidelines (OpenPHACTS Nanopublication Guidelines v1.8.1, 2013 Nanopublication Guidelines PDFs in the design folder) and published nanopublication-style named graphs (e.g. ddi-spl-annotation-np-head-389).  
  Fetch label: fetch:github.com
- The project's published outputs on the Micropublication+Open Annotation approach are workshop papers from 2014-2015 (LISC2014, ArgDiaP 2014, BDM2I 2015 at ISWC), with no later publications listed in the README.  
  Fetch label: fetch:github.com
- As of this 2025 paper, the Micropublications model (Clark et al.) is cited as prior related work but was not adopted or extended; the authors judged it out of scope and note it represents claims in textual form, whereas they wanted machine-readable RDF assertions and instead extended the nanopublication model with a new PROV-O-based 'knowledge provenance' graph.  
  Fetch label: fetch:ceur-ws.org
- The standard three-graph nanopublication model (assertion, provenance, publication info) and PROV-O cannot natively represent supporting versus conflicting evidence or reliability scores for an assertion aggregated from multiple sources; the authors add a fourth named graph ('knowledge provenance') with explicit supportedBy / conflictingWith properties, leaving the original components unchanged for backward compatibility.  
  Fetch label: fetch:ceur-ws.org
- Support and challenge are modelled as evidential relations to specific evidence sentences with probabilistic reliability (assigned certainty degree, ReliableFact/UnreliableFact with InsufficientEvidence or ContrastingEvidence subclasses and explicit unmet reliability conditions), not as social endorsement or votes.  
  Fetch label: fetch:ceur-ws.org
- The extended model was implemented with working software and a large published dataset: the authors extended the fair-workflows 'nanopub' Python package, released source code on GitHub (mntlra/knowledgeProvenance), and published 197,511 extended nanopublications from CoreKB (156,172 reliable facts, 41,339 contrasting-evidence facts) browsable in the CoreKB platform and downloadable from Zenodo.  
  Fetch label: fetch:ceur-ws.org
- The paper's nanopublications combine several active vocabularies for evidence, method, attribution and versioning: PROV-O for the provenance graph, PAV for authorship and versioning, PRV for datasets, the Weighted Evidence (wi:evidence) vocabulary plus ECO (e.g. 'Automatic Assertion', 'Combinatorial Evidence') for evidence annotation, SIO for topic/process, and the nanopubx 'Nanopub Signature Element' for digital signatures; the paper also states that more than 10M nanopublications are publicly accessible.  
  Fetch label: fetch:ceur-ws.org
- The Micropublications ontology is published as an OWL 2 vocabulary at http://purl.org/mp, released under the W3C open source license (the paper does not reference a specific mp.owl version such as v1.18).  
  Fetch label: fetch:ncbi.nlm.nih.gov
- Micropublications were designed to extend, and remain compatible with, statement-based models such as nanopublications and SWAN, which the authors argue cannot model claim networks of arbitrary depth or scientific controversy.  
  Fetch label: fetch:ncbi.nlm.nih.gov
- The model is grounded in Toulmin-Verheij argumentation theory and treats scientific reasoning as defeasible, with a Claim linked via a supportedBy relation to References, Data and other supporting elements, rather than via social endorsement or votes.  
  Fetch label: fetch:ncbi.nlm.nih.gov
- As of 2014, the only software implementation was the Domeo annotation toolkit (Apache 2.0), which implemented an internal version of the model and a Micropublications plugin for Domeo version 2, contextualized using the Open Annotation Model.  
  Fetch label: fetch:ncbi.nlm.nih.gov
- The minimal micropublication is a statement with its attribution, and the model can be added to publications as stand-off metadata serialized in RDF, with provenance handled via SWAN-PAV rather than PROV-O (PROV-O is not mentioned in the paper).  
  Fetch label: fetch:ncbi.nlm.nih.gov
- The mp.owl file served from the timclark001/mp GitHub repository declares owl:versionInfo "1.18" and owl:priorVersion "1.17", but its only dated version note is for 1.17 (May 12, 2016); the ontology declares no versionIRI, no license, no dcterms created/modified dates, and no owl:imports, indicating the artifact has not been versioned or dated since 2016.  
  Fetch label: fetch:raw.githubusercontent.com
- Micropublications is formally built on W3C PROV-O: the ontology reuses prov:Agent, prov:Entity, prov:Person, prov:Organization, and declares mp:hasAttribution as a subproperty of prov:qualifiedAttribution (and of mp:supportedBy), so attribution is modelled as a qualified PROV relation that itself counts as support for the artifact.  
  Fetch label: fetch:raw.githubusercontent.com
- In mp.owl, support and challenge are modelled as logical argument relations rather than endorsements: mp:supports and mp:supportedBy are declared owl:TransitiveProperty, supportedByEvidence is a transitive subproperty of supportedBy with supportedByData and supportedByMethod beneath it, and indirectlyChallenges is defined by the OWL property chain directlyChallenges o supports, so a challenge to supporting evidence propagates to the claim it supports.  
  Fetch label: fetch:raw.githubusercontent.com
- The ontology defines a Micropublication as any Representation that argues some Statement, and a Claim as any Statement that is argued by some Micropublication (via owl:equivalentClass intersections), i.e. a claim is identified by its role as the argued-for statement of a micropublication, with mp:argues declared functional (one claim per micropublication).  
  Fetch label: fetch:raw.githubusercontent.com
- The mp.owl file itself contains no references to nanopublications, Open Annotation/Web Annotation (oa:), SWAN, SEPIO, SIO, FOAF or CiTO; its only external vocabularies are PROV, Dublin Core Terms (dct:bibliographicCitation as superproperty of mp:citation) and SWRL, and it was serialized by the OWL API version 3.4.2 (a 2013-era release).  
  Fetch label: fetch:raw.githubusercontent.com
- SEPIO's latest formal OWL release is dated 2023-06-13 (versionIRI http://purl.obolibrary.org/obo/sepio/releases/2023-06-13/sepio.owl), and that release contains 125 classes, 210 object properties, 32 datatype properties, 71 annotation properties and 21 instances (counts including imports). [Note: bioportal.bioontology.org returned HTTP 403 to both WebFetch and curl; this and the following claims come from the equivalent OBO/Ontobee, OBO Foundry registry and GitHub records for the same ontology.]  
  Fetch label: fetch:bioportal.bioontology.org
- The SEPIO OWL ontology in the monarch-initiative/SEPIO-ontology repository is explicitly declared out of date by its own README, which redirects users to a successor LinkML-based information model and profiling framework at sepio-framework/sepio-linkml; the promised OWL update was targeted for 'fall 2023'.  
  Fetch label: fetch:bioportal.bioontology.org
- SEPIO's core model represents claims, evidence lines, the information items composing those evidence lines, and the methods, tools and agents that created them, with stated use cases of integrating claims with evidence/provenance metadata and supporting evaluation of claims based on that metadata (i.e., evidence is modelled structurally rather than as endorsement counts).  
  Fetch label: fetch:bioportal.bioontology.org
- SEPIO is registered in the OBO Foundry with activity_status 'active', PURL http://purl.obolibrary.org/obo/sepio.owl, CC BY 3.0 license, contact Matthew Brush (ORCID 0000-0002-1048-5019), and is mirrored in Ontobee, AberOWL, OLS, BioPortal and Bioregistry.  
  Fetch label: fetch:bioportal.bioontology.org
- SEPIO's information model is being developed within the GA4GH Variant Annotation (VA) workstream and has a documented ClinGen-ACMG profile for clinical variant pathogenicity interpretations, indicating concrete implementation in clinical genomics data exchange rather than general scientific-claim publishing.  
  Fetch label: fetch:bioportal.bioontology.org
- As of 2026-11-05, the Micropublications namespace PURL with trailing slash (http://purl.org/mp/) does not resolve: purl.org 302-redirects to https://purl.org/mp/, which 307-redirects to https://purl.archive.org/mp/, which returns HTTP 404 Not Found. Term IRIs under that namespace (e.g. http://purl.org/mp/Claim, http://purl.org/mp/argues, http://purl.org/mp/mp.owl) likewise end in 404 at purl.archive.org.  
  Fetch label: fetch:purl.org
- The PURL without a trailing slash (http://purl.org/mp) still resolves (HTTP 200) to the raw ontology file hosted on Tim Clark's personal GitHub account at https://raw.githubusercontent.com/timclark001/mp/master/mp.owl, i.e. the ontology is served from an individual's GitHub repository rather than an institutional or registry host.  
  Fetch label: fetch:purl.org
- The served mp.owl declares itself version 1.18 with prior version 1.17, confirming the version number given in the research question; the file carries no owl:versionIRI and no dated versioning for 1.18.  
  Fetch label: fetch:purl.org
- The most recent date recorded in the ontology header is May 12, 2016 (for version 1.17), with the current 1.18 file carrying no later date, indicating no documented maintenance of the ontology since 2016.  
  Fetch label: fetch:purl.org
- The ontology header attributes authorship to Tim Clark (Massachusetts General Hospital Dept. of Neurology, Harvard Medical School, University of Manchester) with contributors Paolo Ciccarese and Carole Goble, matching the 2014 Journal of Biomedical Semantics paper's authorship.  
  Fetch label: fetch:purl.org
- The Micropublications ontology was published by Clark, Ciccarese and Goble in J Biomed Semantics (2014, 5:28, doi:10.1186/2041-1480-5-28) as an OWL vocabulary resolvable at http://purl.org/mp/, with Ciccarese as primary author of the OWL file; the paper itself gives no version number for mp.owl.  
  Fetch label: fetch:pmc.ncbi.nlm.nih.gov
- The authors position micropublications as distinct from but compatible with nanopublications and BEL: nanopublications model only the indicated statement and are not designed to carry primary evidence, whereas micropublications add an evidence/argument graph while remaining interoperable with nanopublication-coded assertions.  
  Fetch label: fetch:pmc.ncbi.nlm.nih.gov
- In the mp model, support relations form a directed acyclic graph rooted at the micropublication's single Claim; the minimal support for any Statement is attribution to an Agent, and Data may in turn be supported by a Method describing how it was obtained, with challenges inferred from directlyChallenges relations between Representations.  
  Fetch label: fetch:pmc.ncbi.nlm.nih.gov
- The model was implemented in software at publication time only as a plugin to the authors' own Domeo web annotation toolkit (Apache 2.0, version 2), using an internal version of the model, and the authors acknowledged that usefulness depends on widespread adoption and deferred evaluation to a forthcoming article.  
  Fetch label: fetch:pmc.ncbi.nlm.nih.gov
- Micropublications are intended to be serialized as annotation metadata on full-text articles using the W3C Open Annotation model (OAM/AO), and the model was explicitly motivated by shortcomings of the earlier SWAN model (which represented a hypothesis with supporting statements from a single publication).  
  Fetch label: fetch:pmc.ncbi.nlm.nih.gov
