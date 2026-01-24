# Open Hardware/Software IP Protection Guide

A practical guide to protecting your open hardware and software platform while maintaining openness for non-commercial use.

---

## Overview: Three Pillars of Protection

This project employs a comprehensive intellectual property strategy using three complementary forms of legal protection:

### Copyright

Copyright protects the original expression of ideas — your source code, hardware schematics, PCB layouts, CAD designs, and documentation. Copyright exists automatically upon creation but registration provides critical enforcement advantages including statutory damages and attorney's fees. Our dual-license structure (AGPL + CERN-OHL-S with non-commercial exceptions) leverages copyright to ensure commercial users either contribute back or purchase a license.

### Trademark

Trademarks protect brand identity — the project name, logo, and associated goodwill. Even when our open-source licenses permit copying the technology, trademarks prevent others from passing off their products as ours or implying official endorsement. Trademark protection is essential for maintaining brand integrity and preventing market confusion.

### Patent

Patents protect novel inventions and technical innovations — the functional aspects of how something works, not just how it's expressed. While copyright protects code as written, patents can protect the underlying methods and systems regardless of implementation. Patent protection provides the strongest exclusionary rights and can prevent competitors from independently developing similar solutions.

### Why All Three Matter

| Protection | What It Covers | Key Benefit |
|------------|----------------|-------------|
| **Copyright** | Expression (code, designs, docs) | Prevents direct copying |
| **Trademark** | Brand identity (name, logo) | Prevents market confusion |
| **Patent** | Inventions (methods, systems) | Prevents independent development |

Together, these three forms of protection create overlapping layers of defense. A competitor cannot simply rewrite your code to avoid copyright, use a different name to avoid trademarks, or claim independent invention to avoid patents. This comprehensive approach maximizes legal options for enforcement while still permitting non-commercial use under our open licensing terms.

---

## Table of Contents

1. [License Structure](#license-structure)
2. [Copyright Registration](#copyright-registration)
3. [Trademark Protection](#trademark-protection)
4. [Patent Protection](#patent-protection)
5. [Documentation Protocol](#documentation-protocol)

---

## License Structure

### The Strategy: AGPL Base + Non-Commercial Exception

Instead of a non-commercial license that converts to copyleft upon violation (legally untested), use a copyleft license as the base and grant an exception for non-commercial use. This is legally cleaner and well-understood.

### Why This Works

| Scenario | Result |
|----------|--------|
| Hobbyist builds one for personal use | Exempt from copyleft, only attribution required |
| Researcher modifies for academic work | Can keep modifications private (NC use) |
| Company uses commercially, releases source | Legal under AGPL |
| Company uses commercially, keeps source closed | Violating AGPL — you have grounds to sue |
| Company wants closed-source commercial use | Must purchase commercial license from you |

### License Text for Software/Firmware

Create a file called `LICENSE` in your repository:

```
SPDX-License-Identifier: AGPL-3.0-or-later WITH NonCommercial-Exception

Copyright (c) [YEAR] [YOUR COMPANY/NAME]

This work is licensed under the GNU Affero General Public License version 3.0
(AGPL-3.0) with the following additional permission:

ADDITIONAL PERMISSION FOR NON-COMMERCIAL USE

If your use of this work is exclusively non-commercial, you are granted an
additional permission to use, copy, modify, and distribute this work without
the copyleft obligations set forth in AGPL-3.0 sections 4, 5, and 6.

"Non-commercial use" means use that is not primarily intended for or directed
towards commercial advantage or monetary compensation. This includes:
  - Personal hobby projects
  - Academic research and education
  - Non-profit organizational use
  - Evaluation and prototyping (prior to commercial deployment)

Attribution as specified in section 7(b) is still required for all uses.

COMMERCIAL USE

Any commercial use of this work must either:
  1. Comply with the full terms of AGPL-3.0, including the requirement to
     release complete corresponding source code for any modified versions
     or derivative works; OR
  2. Obtain a separate commercial license from [YOUR COMPANY/NAME].

For commercial licensing inquiries, contact: [EMAIL]

THE FULL TEXT OF AGPL-3.0 FOLLOWS:
[Include full AGPL-3.0 text or reference https://www.gnu.org/licenses/agpl-3.0.txt]
```

### License Text for Hardware

Create a file called `HARDWARE-LICENSE` for schematics, PCB designs, and CAD files:

```
SPDX-License-Identifier: CERN-OHL-S-2.0 WITH NonCommercial-Exception

Copyright (c) [YEAR] [YOUR COMPANY/NAME]

This hardware design is licensed under the CERN Open Hardware Licence Version 2
- Strongly Reciprocal (CERN-OHL-S-2.0) with the following additional permission:

ADDITIONAL PERMISSION FOR NON-COMMERCIAL USE

If your use of this design is exclusively non-commercial, you are granted an
additional permission to use, copy, modify, and distribute this design without
the reciprocal licensing obligations set forth in CERN-OHL-S-2.0.

"Non-commercial use" means use that is not primarily intended for or directed
towards commercial advantage or monetary compensation. This includes:
  - Personal hobby projects
  - Academic research and education
  - Non-profit organizational use
  - Evaluation and prototyping (prior to commercial deployment)

Attribution is still required for all uses.

COMMERCIAL USE

Any commercial use of this design must either:
  1. Comply with the full terms of CERN-OHL-S-2.0, including the requirement
     to release complete design files for any modified versions; OR
  2. Obtain a separate commercial license from [YOUR COMPANY/NAME].

For commercial licensing inquiries, contact: [EMAIL]

THE FULL TEXT OF CERN-OHL-S-2.0 FOLLOWS:
[Include full text or reference https://ohwr.org/cern_ohl_s_v2.txt]
```

### README Section

Add this to your README.md:

```markdown
## License

This project uses a dual-license structure:

- **Software/Firmware**: [AGPL-3.0](LICENSE) with Non-Commercial Exception
- **Hardware Designs**: [CERN-OHL-S-2.0](HARDWARE-LICENSE) with Non-Commercial Exception

### Non-Commercial Use (Free)

You may freely use, modify, and share this project for:
- Personal projects
- Academic research and education
- Non-profit use
- Evaluation and prototyping

No copyleft obligations apply to non-commercial use. Just provide attribution.

### Commercial Use

Commercial use requires either:
1. **Full open-source compliance**: Release your complete source code and design files under the same licenses (AGPL-3.0 / CERN-OHL-S-2.0)
2. **Commercial license**: Contact [EMAIL] for pricing

Commercial licenses start at $[X]. We offer startup, business, and enterprise tiers.
```

### Contributor License Agreement (CLA)

To maintain the ability to offer commercial licenses, you must have rights to all contributions. Require contributors to sign a CLA.

Create `CLA.md`:

```markdown
# Contributor License Agreement

Thank you for your interest in contributing to [PROJECT NAME] ("Project").

By submitting a contribution to this Project, you agree to the following terms:

## 1. Definitions

"Contribution" means any original work of authorship, including any modifications
or additions to existing work, that you intentionally submit to this Project.

"Submit" means any form of communication sent to the Project maintainers, including
pull requests, issues, emails, or other electronic communication.

## 2. Grant of Rights

You hereby grant to [YOUR COMPANY/NAME] and recipients of the Project:

a) A perpetual, worldwide, non-exclusive, royalty-free, irrevocable license to
   reproduce, prepare derivative works of, publicly display, publicly perform,
   sublicense, and distribute your Contributions and such derivative works.

b) A perpetual, worldwide, non-exclusive, royalty-free, irrevocable license under
   any patent claims that you own or control to make, use, sell, offer for sale,
   import, and otherwise transfer the Contribution.

## 3. Representations

You represent that:
a) You are the original author of the Contribution
b) You have the legal right to grant the above licenses
c) Your Contribution does not violate any third party's rights

## 4. Commercial Licensing

You acknowledge that [YOUR COMPANY/NAME] may offer commercial licenses for the
Project that include your Contributions, and you will not receive compensation
for such commercial licensing unless separately agreed in writing.

## 5. Attribution

[YOUR COMPANY/NAME] agrees to include your name in the CONTRIBUTORS file or
similar attribution mechanism.
```

For larger projects, consider using a CLA management tool like CLA Assistant (https://cla-assistant.io/).

---

## Copyright Registration

Copyright exists automatically upon creation, but registration provides critical legal advantages. In the US, registration is required before filing an infringement lawsuit.

### Why Register

| Benefit | Unregistered | Registered Before Infringement |
|---------|--------------|-------------------------------|
| Can sue for infringement | Yes (but must register first) | Yes, immediately |
| Actual damages | Yes | Yes |
| Statutory damages ($750-$30,000 per work) | No | Yes |
| Enhanced damages for willful infringement (up to $150,000) | No | Yes |
| Attorney's fees recovery | No | Yes |
| Prima facie evidence of ownership | No | Yes |

**Critical timing**: You must register BEFORE infringement occurs, or within 3 months of first publication, to get statutory damages and attorney's fees.

### What to Register

Register these as separate works:

1. **Software/Firmware** — Register the source code as a literary work
2. **Hardware Designs** — Register schematics, PCB layouts as visual/technical drawings
3. **CAD Files** — Register mechanical designs as visual works or technical drawings
4. **Documentation** — Register user manuals, guides as literary works

### US Copyright Registration Process (USPTO via Copyright.gov)

#### Step 1: Create an Account

1. Go to https://www.copyright.gov/
2. Click "Register a Copyright"
3. Create an account at https://eco.copyright.gov/

#### Step 2: Prepare Your Deposit Materials

For **software**:
- First 25 pages and last 25 pages of source code (if over 50 pages)
- Or entire source code if under 50 pages
- You may redact trade secrets (block out up to 50% of code)
- Include a representative title page or header

For **hardware schematics/PCB layouts**:
- PDF exports of schematics
- Gerber file renders or PDF exports
- Include title block with your name, date, version

For **CAD/mechanical designs**:
- PDF exports or rendered images
- Multiple views (isometric, front, side, exploded)
- Include title block with your name, date, version

For **documentation**:
- Complete PDF of the documentation
- Include title page and copyright notice

#### Step 3: Complete the Application

1. Log into https://eco.copyright.gov/
2. Click "Register a New Claim"
3. Select the type of work:
   - Software → "Literary Work"
   - Schematics/PCB → "Work of the Visual Arts"
   - Documentation → "Literary Work"
4. Fill in required fields:
   - **Title**: Your project name + specific component (e.g., "ProjectName Firmware v1.0")
   - **Year of Completion**: When you finished this version
   - **Date of First Publication**: When you first released it publicly (or leave blank if unpublished)
   - **Author**: Your name or company name
   - **Claimant**: Same as author (or your company if work-for-hire)
5. Upload your deposit materials
6. Pay the fee ($65 for single online application as of 2024)

#### Step 4: Track and Store

- Save your confirmation number
- Registration takes 3-10 months to process
- You'll receive a registration certificate
- Store the certificate securely (digital and physical copies)

#### Step 5: Re-register Major Versions

When you release significant new versions:
- Register the new version as a derivative work
- Reference the original registration number
- This extends protection to new code/designs

### International Considerations

- **Berne Convention**: Your copyright is automatically recognized in 180+ countries
- **No registration required** in most countries for basic protection
- **US registration** is still valuable for US market enforcement
- Consider registration in **China (NCAC)** and **EU** if those are major markets
- China: Register at http://www.ccopyright.com.cn/ (Chinese language, consider using an agent)

### Registration Schedule

| Milestone | Action |
|-----------|--------|
| Before first public release | Register v1.0 of all components |
| Within 3 months of release | Must be registered to preserve statutory damages |
| Each major version (1.0, 2.0, etc.) | Register as derivative work |
| Annually | Review and register any new significant components |

---

## Trademark Protection

Trademarks protect your brand identity — name, logo, tagline. Even if someone legally copies your open-source design, they cannot use your brand.

### What to Trademark

1. **Word mark**: Your project/product name (e.g., "PROJECTNAME")
2. **Logo**: Your graphical logo
3. **Tagline**: Any distinctive tagline (optional)

### Trademark Rights Without Registration

You gain "common law" trademark rights by using a mark in commerce:
- Use ™ symbol (no registration required)
- Limited to geographic areas where you actually do business
- Harder to enforce, must prove you used it first

### Why Register

| Benefit | Unregistered (™) | Registered (®) |
|---------|------------------|----------------|
| Legal presumption of ownership | No | Yes |
| Nationwide priority | No | Yes |
| Listed in USPTO database | No | Yes |
| Use ® symbol | No | Yes |
| Can file in federal court | Difficult | Yes |
| Can record with US Customs (block imports) | No | Yes |
| Basis for international registration | No | Yes |

### US Trademark Registration Process (USPTO)

#### Step 1: Search for Conflicts

Before applying, search for existing similar marks:

1. **USPTO TESS**: https://tess2.uspto.gov/ (free)
   - Search for exact matches
   - Search for phonetic equivalents
   - Search in your product class

2. **Google/domain search**: Check for unregistered users
3. **State databases**: Search state trademark databases
4. **Consider professional search**: ($300-500) for comprehensive clearance

#### Step 2: Determine Your Filing Basis

- **Use in commerce (1a)**: You're already selling products with the mark — fastest
- **Intent to use (1b)**: You plan to use it but haven't yet — requires later proof of use

#### Step 3: Identify Your Class(es)

Trademarks are registered in specific classes. Common ones for hardware/software:

| Class | Description |
|-------|-------------|
| Class 9 | Computer hardware, software, electronic devices |
| Class 42 | Software as a service (SaaS), design services |
| Class 41 | Education, training services |

You can register in multiple classes (additional fee per class).

#### Step 4: File the Application

1. Go to https://www.uspto.gov/trademarks
2. Click "Apply for a trademark"
3. Use TEAS Standard or TEAS Plus application
   - TEAS Plus: $250/class (stricter requirements, pre-approved descriptions)
   - TEAS Standard: $350/class (more flexibility)
4. Fill in:
   - **Mark**: Your name or upload logo image
   - **Owner**: Your name or company (legal entity)
   - **Class**: Select appropriate class(es)
   - **Goods/Services**: Describe what you sell (e.g., "Computer hardware for IoT applications; downloadable firmware for electronic devices")
   - **Specimen**: Proof of use (photo of product with mark, screenshot of website selling product)
   - **Filing basis**: 1(a) or 1(b)
5. Pay fee and submit

#### Step 5: Respond to Office Actions

- Examiner reviews in 3-4 months
- You may receive an "Office Action" requiring clarification or changes
- Respond within 6 months or application is abandoned
- Consider hiring a trademark attorney for Office Actions

#### Step 6: Publication and Opposition

- If approved, mark is published in Official Gazette
- Third parties have 30 days to oppose
- If no opposition, registration proceeds

#### Step 7: Registration and Maintenance

- Receive registration certificate
- **Between years 5-6**: File Section 8 Declaration (proof of continued use)
- **Between years 9-10**: File Section 8 and Section 9 renewal
- **Every 10 years thereafter**: Renew

### Trademark Usage Guidelines

Create a public trademark usage policy:

```markdown
## Trademark Guidelines

"[PROJECTNAME]" and the [PROJECTNAME] logo are trademarks of [YOUR COMPANY].

### Permitted Uses
- Refer to our products by name in articles, reviews, and discussions
- Indicate compatibility (e.g., "Compatible with [PROJECTNAME]")
- Use in academic papers with attribution

### Prohibited Uses
- Using our name/logo to sell products (without license)
- Implying endorsement or affiliation
- Modifying our logo
- Using in domain names for commercial purposes

### Attribution
When referencing [PROJECTNAME], include: "[PROJECTNAME] is a trademark of [YOUR COMPANY]."
```

### International Trademark Registration

For international protection:

1. **Madrid Protocol**: File one application covering multiple countries
   - Done through USPTO after US registration
   - Select desired countries
   - Fees vary by country

2. **Direct filing**: File separately in key markets
   - China: File directly with CNIPA (highly recommended due to trademark squatting)
   - EU: File with EUIPO for EU-wide coverage

**Priority tip**: File in China early, even before product launch. Trademark squatting is common.

---

## Patent Protection

Patents protect the functional aspects of inventions — the novel methods, systems, and processes that make your technology work. Unlike copyright (which protects expression) or trademarks (which protect brand identity), patents protect the underlying innovation itself.

### Why Patents Matter for Open Hardware/Software

| Scenario | Without Patents | With Patents |
|----------|-----------------|--------------|
| Competitor reverse-engineers your design | Legal (if they avoid copying expression) | Potentially infringing |
| Competitor independently develops similar solution | Legal | Potentially infringing |
| Larger company enters market with resources | You compete on execution alone | You have exclusionary rights |
| Licensing negotiations | Limited leverage | Strong negotiating position |
| Acquisition discussions | IP portfolio is weaker | Patents add significant value |

### What Can Be Patented

*[To be expanded with specific patentable innovations]*

### Patent Strategy

*[To be expanded with filing strategy, provisional vs. non-provisional, PCT international filing, etc.]*

### Patent Registration Process

*[To be expanded with USPTO process, timing considerations, and cost estimates]*

### Maintenance and Enforcement

*[To be expanded with maintenance fees, monitoring, and enforcement procedures]*

---

## Documentation Protocol

Thorough documentation creates evidence for legal disputes and establishes prior art.

### Why Documentation Matters

In any IP dispute, you need to prove:
1. You created the work
2. When you created it
3. What you created
4. Your continuous development and use

### What to Document

#### Development Records

| Item | Why It Matters |
|------|----------------|
| Git commit history | Timestamped proof of development progression |
| Design files with dates | Shows evolution of hardware designs |
| Lab notebooks | Traditional evidence of invention (especially for patents) |
| Photos/videos of prototypes | Physical evidence of development |
| Internal communications | Shows decision-making process |
| Meeting notes | Documents team discussions and decisions |

#### Business Records

| Item | Why It Matters |
|------|----------------|
| First sale/shipment date | Establishes commercial use for trademarks |
| Customer invoices | Proves commercial activity |
| Marketing materials | Shows public use of marks |
| Website archives | Documents public presence over time |
| Press coverage | Third-party evidence of your market presence |

#### Publication Records

| Item | Why It Matters |
|------|----------------|
| Release announcements | Public timestamp of each version |
| Archive.org snapshots | Third-party proof of publication dates |
| Hacker News/Reddit posts | Public community record |
| Conference presentations | Public disclosure with witnesses |
| Academic papers | Peer-reviewed timestamp |

### Documentation Procedures

#### 1. Version Control Everything

```bash
# Use git for ALL project files, not just code
git init
git add hardware/    # Schematics, PCB, CAD
git add firmware/    # Source code
git add docs/        # Documentation
git add legal/       # License files

# Use meaningful commit messages with dates
git commit -m "Initial v1.0 release - 2024-01-15"

# Tag releases
git tag -a v1.0 -m "Version 1.0 - first public release"
git push origin v1.0
```

#### 2. Timestamp Critical Documents

**Method 1: Signed Git Commits**
```bash
# Set up GPG signing
git config --global commit.gpgsign true

# All commits are now cryptographically signed with timestamp
```

**Method 2: Archive.org Wayback Machine**
- Submit your project page/releases to https://web.archive.org/save
- Creates independent third-party timestamp
- Do this for each major release

**Method 3: Blockchain Timestamping (optional)**
- Services like OriginStamp, OpenTimestamps
- Creates immutable timestamp proof
- Hash your files and record on blockchain

**Method 4: Self-notarization**
- Create a ZIP of release files
- Generate SHA-256 hash
- Email hash to yourself (creates timestamp in email headers)
- Or use a notary service

#### 3. Maintain a Development Log

Create `CHANGELOG.md` or internal log:

```markdown
# Development Log

## 2024-01-15: Version 1.0 Released
- First public release
- Features: [list]
- Files registered for copyright: [list]
- Git tag: v1.0
- Archive.org snapshot: [URL]

## 2024-02-20: Version 1.1 Development Started
- Adding feature X
- Prototype photos: /docs/photos/2024-02-20/
- Design review meeting notes: /docs/meetings/2024-02-20.md
```

#### 4. Photograph Physical Work

For hardware projects:

```
/evidence/
  /prototypes/
    /2024-01-10_breadboard_v1/
      - photo_1.jpg
      - photo_2.jpg
      - notes.md (what this shows, who witnessed)
    /2024-01-25_pcb_v1/
      - assembled_front.jpg
      - assembled_back.jpg
      - running_demo.mp4
      - notes.md
```

- Include a newspaper or printout with date in photos (old-school but effective)
- Have witnesses sign off on milestone achievements
- Back up to multiple locations (cloud + local)

#### 5. Archive Communications

- Save important emails about design decisions
- Export Slack/Discord discussions about key features
- Document meetings with written summaries

#### 6. Evidence Locker

Create a secure "evidence locker" — a collection of timestamped proof:

```
/evidence_locker/
  /copyright_registrations/
    - certificate_firmware_v1.pdf
    - certificate_hardware_v1.pdf
  /trademark_registrations/
    - certificate_wordmark.pdf
    - certificate_logo.pdf
  /timestamps/
    - v1.0_sha256_hashes.txt
    - v1.0_archive_org_confirmation.pdf
    - v1.0_signed_git_tag.txt
  /prototypes/
    - [dated photo folders]
  /releases/
    - v1.0_release_notes.md
    - v1.0_files.zip
    - v1.0_sha256sum.txt
  /commercial/
    - first_sale_invoice_2024-03-01.pdf
    - trademark_specimen_website_screenshot.png
```

Store copies in:
- Your primary development machine
- Cloud backup (encrypted)
- Physical offline backup (external drive in safe/safety deposit box)
- Optionally: attorney's office

### Regular Documentation Schedule

| Frequency | Task |
|-----------|------|
| Every commit | Meaningful commit message describing changes |
| Every release | Tag in git, changelog entry, archive.org submission |
| Monthly | Backup evidence locker, review documentation completeness |
| Quarterly | Update development log summary |
| Annually | Review IP registrations, file renewals as needed |

---

## Enforcement Checklist

When you discover a potential violation:

### 1. Document the Violation

- [ ] Screenshot/archive the infringing product listing
- [ ] Purchase a sample if possible (keep receipt)
- [ ] Document how it copies your work
- [ ] Record the date you discovered it
- [ ] Identify the seller/manufacturer if possible

### 2. Assess the Situation

- [ ] Are they clearly commercial?
- [ ] Are they complying with AGPL (releasing source)?
- [ ] Are they using your trademark?
- [ ] What jurisdiction are they in?
- [ ] How significant is their sales volume?

### 3. Response Options (escalating)

1. **Reach out directly**: Sometimes it's a misunderstanding; offer commercial license
2. **DMCA takedown**: For online platforms (Amazon, eBay, GitHub, etc.)
3. **Cease and desist letter**: Formal legal demand (have attorney send)
4. **Trademark complaint**: For brand/logo misuse on platforms
5. **Customs recordation**: Block infringing imports at the border (requires registered TM)
6. **Litigation**: File lawsuit (last resort, expensive)

### 4. Platform-Specific Takedowns

**Amazon**: Brand Registry + Report a Violation tool
**eBay**: VeRO program
**Alibaba/AliExpress**: IP Protection Platform (IPP)
**GitHub**: DMCA takedown process
**Shopify**: IP complaint form

---

## Resources and Tools

### Legal

- [AGPL-3.0 Full Text](https://www.gnu.org/licenses/agpl-3.0.txt)
- [CERN-OHL-S-2.0 Full Text](https://ohwr.org/cern_ohl_s_v2.txt)
- [US Copyright Office](https://www.copyright.gov/)
- [USPTO Trademark](https://www.uspto.gov/trademarks)

### Tools

- [CLA Assistant](https://cla-assistant.io/) — Automated CLA management
- [Archive.org](https://web.archive.org/) — Web archiving
- [OpenTimestamps](https://opentimestamps.org/) — Blockchain timestamping

### Professional Help

For significant commercial projects, consider engaging:
- IP attorney (for license drafting, trademark applications, enforcement)
- Patent attorney (if pursuing patents)
- International IP firm (for China, EU registration)

Budget $2,000-$10,000 for initial IP setup with professional help.

---

## Summary Checklist

### Before First Release

- [ ] License files in repository (AGPL + NC exception, CERN-OHL-S + NC exception)
- [ ] README with clear license explanation
- [ ] CLA for contributors
- [ ] Copyright registration filed (within 3 months of publication for full protection)
- [ ] Trademark search completed
- [ ] Trademark application filed
- [ ] Documentation system in place
- [ ] Evidence locker created

### Ongoing

- [ ] Version control all changes
- [ ] Tag and document all releases
- [ ] Archive releases with third-party timestamps
- [ ] Register copyright for major versions
- [ ] Maintain trademark (renewals, monitoring)
- [ ] Monitor for infringement
- [ ] Keep evidence locker backed up

---

*This document is for informational purposes and does not constitute legal advice. Consult with a qualified intellectual property attorney for advice specific to your situation.*
